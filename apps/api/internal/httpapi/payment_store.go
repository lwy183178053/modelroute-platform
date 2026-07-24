package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type SQLPaymentStore struct {
	database *sql.DB
}

func NewSQLPaymentStore(database *sql.DB) *SQLPaymentStore {
	return &SQLPaymentStore{database: database}
}

func (s *SQLPaymentStore) CurrentPaymentSetting(ctx context.Context) (PaymentSetting, error) {
	return scanPaymentSetting(s.database.QueryRowContext(ctx, paymentSettingSelect+" WHERE is_current"))
}

func (s *SQLPaymentStore) CreatePaymentSetting(ctx context.Context, command CreatePaymentSettingCommand) (PaymentSetting, error) {
	methods, err := json.Marshal(command.PaymentMethods)
	if err != nil {
		return PaymentSetting{}, fmt.Errorf("编码支付方式：%w", err)
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return PaymentSetting{}, fmt.Errorf("开始支付配置事务：%w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(782403222)"); err != nil {
		return PaymentSetting{}, fmt.Errorf("锁定支付配置版本：%w", err)
	}
	if _, err := tx.ExecContext(ctx, "UPDATE payment_setting_versions SET is_current = false WHERE is_current"); err != nil {
		return PaymentSetting{}, fmt.Errorf("停用旧支付配置：%w", err)
	}
	setting, err := scanPaymentSetting(tx.QueryRowContext(ctx, `
	INSERT INTO payment_setting_versions
	    (version, gateway_url, merchant_pid, encrypted_key, key_fingerprint, payment_methods,
	     minimum_cny_cents, maximum_cny_cents, withdrawal_tax_bps, enabled, created_by_newapi_user_id)
VALUES
    ((SELECT COALESCE(max(version), 0) + 1 FROM payment_setting_versions),
	     $1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
	RETURNING id, version, gateway_url, merchant_pid, encrypted_key, key_fingerprint, payment_methods,
	          minimum_cny_cents, maximum_cny_cents, withdrawal_tax_bps, enabled, created_at
`, command.GatewayURL, command.MerchantPID, command.EncryptedKey, command.KeyFingerprint, methods,
		command.MinimumCNYCents, command.MaximumCNYCents, command.WithdrawalTaxBPS, command.Enabled, command.AdminUserID))
	if err != nil {
		return PaymentSetting{}, err
	}
	if err := tx.Commit(); err != nil {
		return PaymentSetting{}, fmt.Errorf("提交支付配置事务：%w", err)
	}
	return setting, nil
}

func (s *SQLPaymentStore) CreatePaymentOrder(ctx context.Context, command CreatePaymentOrderCommand) (PaymentOrder, error) {
	return scanPaymentOrder(s.database.QueryRowContext(ctx, `
INSERT INTO payment_orders
    (order_no, newapi_user_id, setting_version_id, payment_type, amount_cny_cents,
     newapi_quota, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING `+paymentOrderColumns, command.OrderNo, command.UserID, command.SettingVersionID,
		command.PaymentType, command.AmountCNYCents, command.NewAPIQuota, command.ExpiresAt))
}

func (s *SQLPaymentStore) PaymentCallbackContext(ctx context.Context, orderNo string) (PaymentCallbackContext, error) {
	row := s.database.QueryRowContext(ctx, `
SELECT `+prefixedPaymentOrderColumns("o")+`,
       p.id, p.version, p.gateway_url, p.merchant_pid, p.encrypted_key, p.key_fingerprint,
	       p.payment_methods, p.minimum_cny_cents, p.maximum_cny_cents, p.withdrawal_tax_bps, p.enabled, p.created_at
FROM payment_orders o
JOIN payment_setting_versions p ON p.id = o.setting_version_id
WHERE o.order_no = $1
`, orderNo)
	order, setting, err := scanPaymentCallbackContext(row)
	return PaymentCallbackContext{Order: order, Setting: setting}, err
}

func (s *SQLPaymentStore) PaymentOrdersForUser(ctx context.Context, userID int, page int, pageSize int) ([]PaymentOrder, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	var total int
	if err := s.database.QueryRowContext(ctx, "SELECT count(*) FROM payment_orders WHERE newapi_user_id = $1", userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计用户充值订单：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `SELECT `+paymentOrderColumns+`
FROM payment_orders
WHERE newapi_user_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询用户充值订单：%w", err)
	}
	items, err := scanPaymentOrders(rows)
	return items, total, err
}

func (s *SQLPaymentStore) PaymentOrderForUser(ctx context.Context, userID int, orderNo string) (PaymentOrder, error) {
	return scanPaymentOrder(s.database.QueryRowContext(ctx, `SELECT `+paymentOrderColumns+`
FROM payment_orders
WHERE newapi_user_id = $1 AND order_no = $2`, userID, orderNo))
}

func (s *SQLPaymentStore) ListPaymentOrders(ctx context.Context, page int, pageSize int, status string) ([]PaymentOrder, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	var total int
	if err := s.database.QueryRowContext(ctx, `
SELECT count(*) FROM payment_orders
WHERE ($1 = '' OR payment_status = $1 OR credit_status = $1)`, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计充值订单：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `SELECT `+paymentOrderColumns+`
FROM payment_orders
WHERE ($1 = '' OR payment_status = $1 OR credit_status = $1)
ORDER BY id DESC
LIMIT $2 OFFSET $3`, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询充值订单：%w", err)
	}
	items, err := scanPaymentOrders(rows)
	return items, total, err
}

func (s *SQLPaymentStore) AcceptPaymentCallback(ctx context.Context, command AcceptPaymentCallbackCommand) (PaymentOrder, bool, error) {
	payload, err := json.Marshal(command.Payload)
	if err != nil {
		return PaymentOrder{}, false, fmt.Errorf("编码支付回调：%w", err)
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return PaymentOrder{}, false, fmt.Errorf("开始支付回调事务：%w", err)
	}
	defer tx.Rollback()
	order, err := scanPaymentOrder(tx.QueryRowContext(ctx, `SELECT `+paymentOrderColumns+`
FROM payment_orders WHERE order_no = $1 FOR UPDATE`, command.OrderNo))
	if err != nil {
		return PaymentOrder{}, false, err
	}
	if order.PaymentStatus == "paid" {
		if order.ThirdPartyTradeNo != command.TradeNo {
			return PaymentOrder{}, false, ErrPaymentConflict
		}
		if err := tx.Commit(); err != nil {
			return PaymentOrder{}, false, fmt.Errorf("提交重复回调事务：%w", err)
		}
		return order, false, nil
	}
	if order.PaymentStatus != "pending" {
		return PaymentOrder{}, false, ErrPaymentConflict
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO payment_callback_events
    (callback_key, payment_order_id, third_party_trade_no, provider_payload, received_at)
VALUES ($1, $2, $3, $4::jsonb, $5)
`, command.CallbackKey, order.ID, command.TradeNo, payload, command.PaidAt); err != nil {
		return PaymentOrder{}, false, paymentStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE payment_orders
SET payment_status = 'paid', third_party_trade_no = $1, paid_at = $2, updated_at = now()
WHERE id = $3
`, command.TradeNo, command.PaidAt, order.ID); err != nil {
		return PaymentOrder{}, false, paymentStoreError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO newapi_credit_operations (payment_order_id, idempotency_key)
VALUES ($1, $2)
`, order.ID, "payment:"+order.OrderNo+":credit"); err != nil {
		return PaymentOrder{}, false, paymentStoreError(err)
	}
	if err := tx.Commit(); err != nil {
		return PaymentOrder{}, false, paymentStoreError(err)
	}
	order.PaymentStatus = "paid"
	order.ThirdPartyTradeNo = command.TradeNo
	order.PaidAt = &command.PaidAt
	return order, true, nil
}

func (s *SQLPaymentStore) MarkStalePaymentCreditsUnknown(ctx context.Context, before time.Time) (int, error) {
	result, err := s.database.ExecContext(ctx, `
WITH stale AS (
    UPDATE newapi_credit_operations
    SET status = 'unknown', last_error = '入账请求处理中断，结果需要人工核对',
        finished_at = now(), updated_at = now()
    WHERE status = 'processing' AND started_at < $1
    RETURNING payment_order_id
)
UPDATE payment_orders
SET credit_status = 'unknown', updated_at = now()
WHERE id IN (SELECT payment_order_id FROM stale)
`, before)
	if err != nil {
		return 0, fmt.Errorf("标记超时入账：%w", err)
	}
	count, err := result.RowsAffected()
	return int(count), err
}

func (s *SQLPaymentStore) ClaimPaymentCredits(ctx context.Context, now time.Time, limit int) ([]PaymentCreditOperation, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开始领取入账事务：%w", err)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
SELECT c.id, c.payment_order_id, o.order_no, o.newapi_user_id, o.newapi_quota, c.attempt_count
FROM newapi_credit_operations c
JOIN payment_orders o ON o.id = c.payment_order_id
WHERE c.status IN ('pending', 'failed')
  AND c.attempt_count < 5
  AND c.next_attempt_at <= $1
  AND o.payment_status = 'paid'
ORDER BY c.id
FOR UPDATE OF c SKIP LOCKED
LIMIT $2
`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("查询待入账任务：%w", err)
	}
	operations := make([]PaymentCreditOperation, 0)
	for rows.Next() {
		var operation PaymentCreditOperation
		if err := rows.Scan(&operation.ID, &operation.OrderID, &operation.OrderNo, &operation.UserID, &operation.NewAPIQuota, &operation.Attempts); err != nil {
			rows.Close()
			return nil, fmt.Errorf("读取待入账任务：%w", err)
		}
		operations = append(operations, operation)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("关闭入账任务结果：%w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历待入账任务：%w", err)
	}
	for index := range operations {
		if _, err := tx.ExecContext(ctx, `
UPDATE newapi_credit_operations
SET status = 'processing', attempt_count = attempt_count + 1, started_at = $1,
    finished_at = NULL, last_error = NULL, updated_at = now()
WHERE id = $2
`, now, operations[index].ID); err != nil {
			return nil, fmt.Errorf("锁定入账任务：%w", err)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE payment_orders SET credit_status = 'processing', updated_at = now() WHERE id = $1
`, operations[index].OrderID); err != nil {
			return nil, fmt.Errorf("更新订单入账状态：%w", err)
		}
		operations[index].Attempts++
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交领取入账事务：%w", err)
	}
	return operations, nil
}

func (s *SQLPaymentStore) CompletePaymentCredit(ctx context.Context, operationID int64, completedAt time.Time) error {
	return s.finishPaymentCredit(ctx, operationID, "succeeded", "", completedAt, completedAt)
}

func (s *SQLPaymentStore) FailPaymentCredit(ctx context.Context, operationID int64, status string, message string, nextAttempt time.Time) error {
	if status != "failed" && status != "unknown" {
		return ErrPaymentConflict
	}
	return s.finishPaymentCredit(ctx, operationID, status, message, time.Now().UTC(), nextAttempt)
}

func (s *SQLPaymentStore) RetryPaymentCredit(ctx context.Context, orderID int64, allowUnknown bool) error {
	expectedStatus := "failed"
	if allowUnknown {
		expectedStatus = "unknown"
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始重试入账事务：%w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
UPDATE newapi_credit_operations
SET status = 'pending', attempt_count = 0, last_error = NULL, next_attempt_at = now(),
    started_at = NULL, finished_at = NULL, updated_at = now()
WHERE payment_order_id = $1 AND status = $2
`, orderID, expectedStatus)
	if err != nil {
		return fmt.Errorf("重新排队入账任务：%w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return ErrPaymentConflict
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE payment_orders SET credit_status = 'pending', credited_at = NULL, updated_at = now() WHERE id = $1
`, orderID); err != nil {
		return fmt.Errorf("更新重试订单：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交重试入账事务：%w", err)
	}
	return nil
}

func (s *SQLPaymentStore) ConfirmPaymentCredit(ctx context.Context, orderID int64, confirmedAt time.Time) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始确认入账事务：%w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
UPDATE newapi_credit_operations
SET status = 'succeeded', last_error = NULL, finished_at = $1, updated_at = now()
WHERE payment_order_id = $2 AND status = 'unknown'
`, confirmedAt, orderID)
	if err != nil {
		return fmt.Errorf("确认入账任务：%w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return ErrPaymentConflict
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE payment_orders
SET credit_status = 'succeeded', credited_at = $1, updated_at = now()
WHERE id = $2
`, confirmedAt, orderID); err != nil {
		return fmt.Errorf("确认订单入账：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交确认入账事务：%w", err)
	}
	return nil
}

func (s *SQLPaymentStore) finishPaymentCredit(ctx context.Context, operationID int64, status string, message string, finishedAt time.Time, nextAttempt time.Time) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("开始完成入账事务：%w", err)
	}
	defer tx.Rollback()
	var orderID int64
	err = tx.QueryRowContext(ctx, `
UPDATE newapi_credit_operations
SET status = $1, last_error = NULLIF($2, ''), next_attempt_at = $3,
    finished_at = $4, updated_at = now()
WHERE id = $5 AND status = 'processing'
RETURNING payment_order_id
`, status, truncatePaymentError(message), nextAttempt, finishedAt, operationID).Scan(&orderID)
	if err != nil {
		return paymentStoreError(err)
	}
	if status == "succeeded" {
		_, err = tx.ExecContext(ctx, `
UPDATE payment_orders
SET credit_status = 'succeeded', credited_at = $1, updated_at = now()
WHERE id = $2
`, finishedAt, orderID)
	} else {
		_, err = tx.ExecContext(ctx, `
UPDATE payment_orders SET credit_status = $1, updated_at = now() WHERE id = $2
`, status, orderID)
	}
	if err != nil {
		return fmt.Errorf("更新订单入账结果：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交完成入账事务：%w", err)
	}
	return nil
}

func truncatePaymentError(message string) string {
	if len(message) <= 500 {
		return message
	}
	return message[:500]
}

const paymentSettingSelect = `
SELECT id, version, gateway_url, merchant_pid, encrypted_key, key_fingerprint, payment_methods,
	   minimum_cny_cents, maximum_cny_cents, withdrawal_tax_bps, enabled, created_at
FROM payment_setting_versions`

const paymentOrderColumns = `id, order_no, newapi_user_id, setting_version_id, payment_type,
       amount_cny_cents, newapi_quota,
       payment_status, credit_status, third_party_trade_no, paid_at, credited_at,
       expires_at, created_at, updated_at`

func prefixedPaymentOrderColumns(alias string) string {
	columns := []string{
		"id", "order_no", "newapi_user_id", "setting_version_id", "payment_type", "amount_cny_cents",
		"newapi_quota", "payment_status", "credit_status",
		"third_party_trade_no", "paid_at", "credited_at", "expires_at", "created_at", "updated_at",
	}
	result := ""
	for index, column := range columns {
		if index > 0 {
			result += ", "
		}
		result += alias + "." + column
	}
	return result
}

func scanPaymentSetting(row rowScanner) (PaymentSetting, error) {
	var setting PaymentSetting
	var methods []byte
	err := row.Scan(&setting.ID, &setting.Version, &setting.GatewayURL, &setting.MerchantPID,
		&setting.EncryptedKey, &setting.KeyFingerprint, &methods, &setting.MinimumCNYCents,
		&setting.MaximumCNYCents, &setting.WithdrawalTaxBPS, &setting.Enabled, &setting.CreatedAt)
	if err != nil {
		return PaymentSetting{}, paymentStoreError(err)
	}
	if err := json.Unmarshal(methods, &setting.PaymentMethods); err != nil {
		return PaymentSetting{}, fmt.Errorf("解析支付方式：%w", err)
	}
	return setting, nil
}

func scanPaymentOrder(row rowScanner) (PaymentOrder, error) {
	var order PaymentOrder
	var tradeNo sql.NullString
	var paidAt, creditedAt sql.NullTime
	err := row.Scan(&order.ID, &order.OrderNo, &order.UserID, &order.SettingVersionID, &order.PaymentType,
		&order.AmountCNYCents, &order.NewAPIQuota,
		&order.PaymentStatus, &order.CreditStatus, &tradeNo, &paidAt, &creditedAt,
		&order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return PaymentOrder{}, paymentStoreError(err)
	}
	if tradeNo.Valid {
		order.ThirdPartyTradeNo = tradeNo.String
	}
	if paidAt.Valid {
		order.PaidAt = &paidAt.Time
	}
	if creditedAt.Valid {
		order.CreditedAt = &creditedAt.Time
	}
	return order, nil
}

func scanPaymentOrders(rows *sql.Rows) ([]PaymentOrder, error) {
	defer rows.Close()
	items := make([]PaymentOrder, 0)
	for rows.Next() {
		item, err := scanPaymentOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历充值订单：%w", err)
	}
	return items, nil
}

func scanPaymentCallbackContext(row rowScanner) (PaymentOrder, PaymentSetting, error) {
	var order PaymentOrder
	var setting PaymentSetting
	var tradeNo sql.NullString
	var paidAt, creditedAt sql.NullTime
	var methods []byte
	err := row.Scan(&order.ID, &order.OrderNo, &order.UserID, &order.SettingVersionID, &order.PaymentType,
		&order.AmountCNYCents, &order.NewAPIQuota,
		&order.PaymentStatus, &order.CreditStatus, &tradeNo, &paidAt, &creditedAt,
		&order.ExpiresAt, &order.CreatedAt, &order.UpdatedAt,
		&setting.ID, &setting.Version, &setting.GatewayURL, &setting.MerchantPID, &setting.EncryptedKey,
		&setting.KeyFingerprint, &methods, &setting.MinimumCNYCents, &setting.MaximumCNYCents,
		&setting.WithdrawalTaxBPS, &setting.Enabled, &setting.CreatedAt)
	if err != nil {
		return PaymentOrder{}, PaymentSetting{}, paymentStoreError(err)
	}
	if tradeNo.Valid {
		order.ThirdPartyTradeNo = tradeNo.String
	}
	if paidAt.Valid {
		order.PaidAt = &paidAt.Time
	}
	if creditedAt.Valid {
		order.CreditedAt = &creditedAt.Time
	}
	if err := json.Unmarshal(methods, &setting.PaymentMethods); err != nil {
		return PaymentOrder{}, PaymentSetting{}, fmt.Errorf("解析支付方式：%w", err)
	}
	return order, setting, nil
}

func paymentStoreError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPaymentNotFound
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && (pgError.Code == "23505" || pgError.Code == "23514" || pgError.Code == "23503") {
		return ErrPaymentConflict
	}
	return err
}
