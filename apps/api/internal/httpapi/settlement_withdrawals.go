package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const merchantWithdrawalColumns = `
id, withdrawal_no, merchant_profile_id, amount_quota, tax_quota, net_payout_quota, available_before_quota,
recipient_type, encrypted_recipient_account, recipient_account_masked, status,
COALESCE(review_reason, ''), COALESCE(reviewed_by_newapi_user_id, 0), reviewed_at,
COALESCE(payout_reference, ''), COALESCE(payout_proof, ''),
COALESCE(paid_by_newapi_user_id, 0), paid_at, created_at, updated_at`

func (s *SQLSettlementStore) MerchantBalance(ctx context.Context, userID int) (MerchantBalance, error) {
	var balance MerchantBalance
	err := s.database.QueryRowContext(ctx, `
SELECT mp.id, COALESCE(b.available_quota, 0), COALESCE(b.frozen_quota, 0),
       COALESCE(b.lifetime_earned_quota, 0), COALESCE(b.lifetime_refunded_quota, 0),
       COALESCE(b.lifetime_paid_quota, 0), COALESCE(b.updated_at, mp.updated_at)
FROM merchant_profiles mp
LEFT JOIN merchant_balances b ON b.merchant_profile_id = mp.id
WHERE mp.newapi_user_id = $1 AND mp.status = 'active'
`, userID).Scan(
		&balance.MerchantID, &balance.AvailableQuota, &balance.FrozenQuota,
		&balance.LifetimeEarnedQuota, &balance.LifetimeRefundedQuota,
		&balance.LifetimePaidQuota, &balance.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantBalance{}, ErrSettlementNotFound
	}
	if err != nil {
		return MerchantBalance{}, fmt.Errorf("读取商户余额：%w", err)
	}
	return balance, nil
}

func (s *SQLSettlementStore) MerchantLedger(ctx context.Context, userID int, page int, pageSize int) ([]MerchantLedgerEntry, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	var merchantID int64
	if err := s.database.QueryRowContext(ctx, `
SELECT id FROM merchant_profiles WHERE newapi_user_id = $1 AND status = 'active'
`, userID).Scan(&merchantID); errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrSettlementNotFound
	} else if err != nil {
		return nil, 0, fmt.Errorf("读取商户账本归属：%w", err)
	}
	var total int
	if err := s.database.QueryRowContext(ctx, `
SELECT count(*) FROM merchant_ledger_entries WHERE merchant_profile_id = $1
`, merchantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计商户账本：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT l.id, l.merchant_profile_id, l.entry_type, l.available_delta, l.frozen_delta,
       l.available_after, l.frozen_after, l.description,
       COALESCE(s.newapi_log_id, 0), COALESCE(s.source_type, 0), COALESCE(cm.public_name, ''),
       COALESCE(s.gross_quota, 0), COALESCE(w.withdrawal_no, ''), l.created_at
FROM merchant_ledger_entries l
LEFT JOIN settlement_sources s ON s.id = l.settlement_source_id
LEFT JOIN offer_versions o ON o.id = s.offer_version_id
LEFT JOIN canonical_models cm ON cm.id = o.canonical_model_id
LEFT JOIN merchant_withdrawals w ON w.id = l.withdrawal_id
WHERE l.merchant_profile_id = $1
ORDER BY l.id DESC LIMIT $2 OFFSET $3
`, merchantID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询商户账本：%w", err)
	}
	defer rows.Close()
	entries := make([]MerchantLedgerEntry, 0)
	for rows.Next() {
		var entry MerchantLedgerEntry
		if err := rows.Scan(
			&entry.ID, &entry.MerchantID, &entry.EntryType, &entry.AvailableDelta,
			&entry.FrozenDelta, &entry.AvailableAfter, &entry.FrozenAfter, &entry.Description,
			&entry.NewAPILogID, &entry.SourceType, &entry.ModelName, &entry.GrossQuota,
			&entry.WithdrawalNo, &entry.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("读取商户账本：%w", err)
		}
		entries = append(entries, entry)
	}
	return entries, total, rows.Err()
}

func (s *SQLSettlementStore) MerchantWithdrawals(ctx context.Context, userID int, page int, pageSize int) ([]MerchantWithdrawal, int, error) {
	return s.listWithdrawals(ctx, page, pageSize, "", userID)
}

func (s *SQLSettlementStore) ListMerchantWithdrawals(ctx context.Context, page int, pageSize int, status string) ([]MerchantWithdrawal, int, error) {
	return s.listWithdrawals(ctx, page, pageSize, status, 0)
}

func (s *SQLSettlementStore) listWithdrawals(ctx context.Context, page int, pageSize int, status string, userID int) ([]MerchantWithdrawal, int, error) {
	page, pageSize = boundedPagination(page, pageSize)
	var total int
	if err := s.database.QueryRowContext(ctx, `
SELECT count(*)
FROM merchant_withdrawals w
JOIN merchant_profiles mp ON mp.id = w.merchant_profile_id
WHERE ($1 = 0 OR mp.newapi_user_id = $1) AND ($2 = '' OR w.status = $2)
`, userID, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计提现申请：%w", err)
	}
	rows, err := s.database.QueryContext(ctx, `
SELECT `+prefixedWithdrawalColumns("w")+`, mp.display_name
FROM merchant_withdrawals w
JOIN merchant_profiles mp ON mp.id = w.merchant_profile_id
WHERE ($1 = 0 OR mp.newapi_user_id = $1) AND ($2 = '' OR w.status = $2)
ORDER BY w.id DESC LIMIT $3 OFFSET $4
`, userID, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("查询提现申请：%w", err)
	}
	defer rows.Close()
	items := make([]MerchantWithdrawal, 0)
	for rows.Next() {
		item, err := scanWithdrawalWithName(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (s *SQLSettlementStore) CreateMerchantWithdrawal(ctx context.Context, command CreateMerchantWithdrawalCommand) (MerchantWithdrawal, error) {
	if _, _, ok := withdrawalBreakdown(command.AmountQuota, command.QuotaPerUnit); command.UserID <= 0 || command.WithdrawalNo == "" || !ok || command.RecipientType != "alipay" || len(command.EncryptedRecipientAccount) == 0 || command.RecipientAccountMasked == "" {
		return MerchantWithdrawal{}, ErrSettlementConflict
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("开始提现事务：%w", err)
	}
	defer tx.Rollback()
	var merchantID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM merchant_profiles WHERE newapi_user_id = $1 AND status = 'active'
`, command.UserID).Scan(&merchantID)
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantWithdrawal{}, ErrSettlementNotFound
	}
	if err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("读取提现商户：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO merchant_balances (merchant_profile_id) VALUES ($1) ON CONFLICT DO NOTHING`, merchantID); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("初始化提现余额：%w", err)
	}
	var availableBefore, frozenBefore int64
	if err := tx.QueryRowContext(ctx, `
SELECT available_quota, frozen_quota FROM merchant_balances WHERE merchant_profile_id = $1 FOR UPDATE
`, merchantID).Scan(&availableBefore, &frozenBefore); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("锁定提现余额：%w", err)
	}
	if availableBefore < command.AmountQuota {
		return MerchantWithdrawal{}, ErrInsufficientMerchantBalance
	}
	withdrawal, err := scanWithdrawal(tx.QueryRowContext(ctx, `
INSERT INTO merchant_withdrawals
    (withdrawal_no, merchant_profile_id, amount_quota, available_before_quota,
     recipient_type, encrypted_recipient_account, recipient_account_masked, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
RETURNING `+merchantWithdrawalColumns,
		command.WithdrawalNo, merchantID, command.AmountQuota, availableBefore, command.RecipientType,
		command.EncryptedRecipientAccount, command.RecipientAccountMasked, command.CreatedAt))
	if err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("创建提现申请：%w", err)
	}
	availableAfter := availableBefore - command.AmountQuota
	frozenAfter := frozenBefore + command.AmountQuota
	if _, err := tx.ExecContext(ctx, `
UPDATE merchant_balances SET available_quota = $2, frozen_quota = $3, updated_at = now()
WHERE merchant_profile_id = $1
`, merchantID, availableAfter, frozenAfter); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("冻结提现余额：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_ledger_entries
    (merchant_profile_id, withdrawal_id, entry_type, available_delta, frozen_delta,
     available_after, frozen_after, description)
VALUES ($1, $2, 'withdrawal_freeze', $3, $4, $5, $6, '提交人工提现并冻结余额')
`, merchantID, withdrawal.ID, -command.AmountQuota, command.AmountQuota, availableAfter, frozenAfter); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("写入提现冻结账本：%w", err)
	}
	if err := tx.Commit(); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("提交提现事务：%w", err)
	}
	return withdrawal, nil
}

func (s *SQLSettlementStore) ApproveMerchantWithdrawal(ctx context.Context, id int64, adminID int, reason string, reviewedAt time.Time) (MerchantWithdrawal, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	defer tx.Rollback()
	withdrawal, err := lockWithdrawal(ctx, tx, id)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	if withdrawal.Status != "pending" {
		return MerchantWithdrawal{}, ErrSettlementConflict
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE merchant_withdrawals
SET status = 'approved', review_reason = $2, reviewed_by_newapi_user_id = $3,
    reviewed_at = $4, updated_at = now() WHERE id = $1
`, id, reason, adminID, reviewedAt); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("审核提现申请：%w", err)
	}
	withdrawal.Status = "approved"
	withdrawal.ReviewReason = reason
	withdrawal.ReviewedByUserID = adminID
	withdrawal.ReviewedAt = &reviewedAt
	if err := tx.Commit(); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("提交提现审核：%w", err)
	}
	return withdrawal, nil
}

func (s *SQLSettlementStore) RejectMerchantWithdrawal(ctx context.Context, id int64, adminID int, reason string, reviewedAt time.Time) (MerchantWithdrawal, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	defer tx.Rollback()
	withdrawal, err := lockWithdrawal(ctx, tx, id)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	if withdrawal.Status != "pending" && withdrawal.Status != "approved" {
		return MerchantWithdrawal{}, ErrSettlementConflict
	}
	available, frozen, err := updateMerchantBalance(ctx, tx, withdrawal.MerchantID, withdrawal.AmountQuota, -withdrawal.AmountQuota, 0, 0, 0)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE merchant_withdrawals
SET status = 'rejected', review_reason = $2, reviewed_by_newapi_user_id = $3,
    reviewed_at = $4, updated_at = now() WHERE id = $1
`, id, reason, adminID, reviewedAt); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("驳回提现申请：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_ledger_entries
    (merchant_profile_id, withdrawal_id, entry_type, available_delta, frozen_delta,
     available_after, frozen_after, description)
VALUES ($1, $2, 'withdrawal_release', $3, $4, $5, $6, '提现驳回并释放冻结余额')
`, withdrawal.MerchantID, withdrawal.ID, withdrawal.AmountQuota, -withdrawal.AmountQuota, available, frozen); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("写入提现释放账本：%w", err)
	}
	withdrawal.Status = "rejected"
	withdrawal.ReviewReason = reason
	withdrawal.ReviewedByUserID = adminID
	withdrawal.ReviewedAt = &reviewedAt
	if err := tx.Commit(); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("提交提现驳回：%w", err)
	}
	return withdrawal, nil
}

func (s *SQLSettlementStore) PayMerchantWithdrawal(ctx context.Context, id int64, adminID int, reference string, proof string, paidAt time.Time) (MerchantWithdrawal, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	defer tx.Rollback()
	withdrawal, err := lockWithdrawal(ctx, tx, id)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	if withdrawal.Status != "approved" {
		return MerchantWithdrawal{}, ErrSettlementConflict
	}
	available, frozen, err := updateMerchantBalance(ctx, tx, withdrawal.MerchantID, 0, -withdrawal.AmountQuota, 0, 0, withdrawal.AmountQuota)
	if err != nil {
		return MerchantWithdrawal{}, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE merchant_withdrawals
SET status = 'paid', payout_reference = $2, payout_proof = $3,
    paid_by_newapi_user_id = $4, paid_at = $5, updated_at = now() WHERE id = $1
`, id, reference, proof, adminID, paidAt); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("确认提现打款：%w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO merchant_ledger_entries
    (merchant_profile_id, withdrawal_id, entry_type, available_delta, frozen_delta,
     available_after, frozen_after, description)
VALUES ($1, $2, 'withdrawal_paid', 0, $3, $4, $5, '管理员确认线下打款')
`, withdrawal.MerchantID, withdrawal.ID, -withdrawal.AmountQuota, available, frozen); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("写入提现出账账本：%w", err)
	}
	withdrawal.Status = "paid"
	withdrawal.PayoutReference = reference
	withdrawal.PayoutProof = proof
	withdrawal.PaidByUserID = adminID
	withdrawal.PaidAt = &paidAt
	if err := tx.Commit(); err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("提交提现打款：%w", err)
	}
	return withdrawal, nil
}

func lockWithdrawal(ctx context.Context, tx *sql.Tx, id int64) (MerchantWithdrawal, error) {
	withdrawal, err := scanWithdrawal(tx.QueryRowContext(ctx, `
SELECT `+merchantWithdrawalColumns+` FROM merchant_withdrawals WHERE id = $1 FOR UPDATE
`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return MerchantWithdrawal{}, ErrSettlementNotFound
	}
	if err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("锁定提现申请：%w", err)
	}
	return withdrawal, nil
}

type withdrawalScanner interface {
	Scan(...any) error
}

func scanWithdrawal(scanner withdrawalScanner) (MerchantWithdrawal, error) {
	var item MerchantWithdrawal
	err := scanner.Scan(
		&item.ID, &item.WithdrawalNo, &item.MerchantID, &item.AmountQuota,
		&item.TaxQuota, &item.NetPayoutQuota, &item.AvailableBeforeQuota, &item.RecipientType, &item.EncryptedRecipientAccount,
		&item.RecipientAccountMasked, &item.Status, &item.ReviewReason, &item.ReviewedByUserID,
		&item.ReviewedAt, &item.PayoutReference, &item.PayoutProof, &item.PaidByUserID,
		&item.PaidAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanWithdrawalWithName(scanner withdrawalScanner) (MerchantWithdrawal, error) {
	var item MerchantWithdrawal
	err := scanner.Scan(
		&item.ID, &item.WithdrawalNo, &item.MerchantID, &item.AmountQuota,
		&item.TaxQuota, &item.NetPayoutQuota, &item.AvailableBeforeQuota, &item.RecipientType, &item.EncryptedRecipientAccount,
		&item.RecipientAccountMasked, &item.Status, &item.ReviewReason, &item.ReviewedByUserID,
		&item.ReviewedAt, &item.PayoutReference, &item.PayoutProof, &item.PaidByUserID,
		&item.PaidAt, &item.CreatedAt, &item.UpdatedAt, &item.MerchantName,
	)
	if err != nil {
		return MerchantWithdrawal{}, fmt.Errorf("读取提现申请：%w", err)
	}
	return item, nil
}

func prefixedWithdrawalColumns(prefix string) string {
	return prefix + `.id, ` + prefix + `.withdrawal_no, ` + prefix + `.merchant_profile_id, ` +
		prefix + `.amount_quota, ` + prefix + `.tax_quota, ` + prefix + `.net_payout_quota, ` + prefix + `.available_before_quota, ` + prefix + `.recipient_type, ` +
		prefix + `.encrypted_recipient_account, ` + prefix + `.recipient_account_masked, ` + prefix + `.status, ` +
		`COALESCE(` + prefix + `.review_reason, ''), COALESCE(` + prefix + `.reviewed_by_newapi_user_id, 0), ` + prefix + `.reviewed_at, ` +
		`COALESCE(` + prefix + `.payout_reference, ''), COALESCE(` + prefix + `.payout_proof, ''), ` +
		`COALESCE(` + prefix + `.paid_by_newapi_user_id, 0), ` + prefix + `.paid_at, ` + prefix + `.created_at, ` + prefix + `.updated_at`
}
