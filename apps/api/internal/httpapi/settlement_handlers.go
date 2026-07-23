package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var withdrawalRecipientTypes = map[string]bool{
	"alipay": true,
	"bank":   true,
	"other":  true,
}

func (s *server) merchantBalance(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.settlementReady(w) {
		return
	}
	balance, err := s.services.Settlement.MerchantBalance(r.Context(), session.UserID)
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": balance})
}

func (s *server) merchantWithdrawals(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.settlementReady(w) {
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Settlement.MerchantWithdrawals(r.Context(), session.UserID, page, pageSize)
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"page": page, "page_size": pageSize, "total": total, "items": items,
	}})
}

func (s *server) createMerchantWithdrawal(w http.ResponseWriter, r *http.Request) {
	_, session, _, ok := s.requireMerchant(w, r)
	if !ok || !s.requireSameOrigin(w, r) || !s.settlementReady(w) {
		return
	}
	var request struct {
		AmountQuota      int64  `json:"amount_quota"`
		RecipientType    string `json:"recipient_type"`
		RecipientAccount string `json:"recipient_account"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal", "提现参数无效")
		return
	}
	request.RecipientType = strings.TrimSpace(request.RecipientType)
	request.RecipientAccount = strings.TrimSpace(request.RecipientAccount)
	if request.AmountQuota <= 0 || !withdrawalRecipientTypes[request.RecipientType] || len(request.RecipientAccount) < 5 || len(request.RecipientAccount) > 256 {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal", "提现金额或收款账户无效")
		return
	}
	accountBytes := []byte(request.RecipientAccount)
	encrypted, err := s.services.Credentials.Encrypt(accountBytes)
	zeroBytes(accountBytes)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "withdrawal_encrypt_failed", "加密收款账户失败")
		return
	}
	now := time.Now().UTC()
	withdrawalNo, err := newWithdrawalNo(now)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "withdrawal_number_failed", "生成提现编号失败")
		return
	}
	withdrawal, err := s.services.Settlement.CreateMerchantWithdrawal(r.Context(), CreateMerchantWithdrawalCommand{
		WithdrawalNo: withdrawalNo, UserID: session.UserID, AmountQuota: request.AmountQuota,
		RecipientType: request.RecipientType, EncryptedRecipientAccount: encrypted,
		RecipientAccountMasked: maskRecipientAccount(request.RecipientAccount), CreatedAt: now,
	})
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "data": withdrawal})
}

func (s *server) adminWithdrawals(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireSession(w, r, roleAdmin); !ok || !s.settlementReady(w) {
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !map[string]bool{"pending": true, "approved": true, "paid": true, "rejected": true}[status] {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_status", "提现状态筛选无效")
		return
	}
	page, pageSize := pagination(r)
	items, total, err := s.services.Settlement.ListMerchantWithdrawals(r.Context(), page, pageSize, status)
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	for index := range items {
		plaintext, err := s.services.Credentials.Decrypt(items[index].EncryptedRecipientAccount)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "withdrawal_decrypt_failed", "读取收款账户失败")
			return
		}
		items[index].RecipientAccount = string(plaintext)
		zeroBytes(plaintext)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"page": page, "page_size": pageSize, "total": total, "items": items,
	}})
}

func (s *server) adminApproveWithdrawal(w http.ResponseWriter, r *http.Request) {
	s.adminReviewWithdrawal(w, r, "approve")
}

func (s *server) adminRejectWithdrawal(w http.ResponseWriter, r *http.Request) {
	s.adminReviewWithdrawal(w, r, "reject")
}

func (s *server) adminReviewWithdrawal(w http.ResponseWriter, r *http.Request, action string) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.settlementReady(w) {
		return
	}
	id, ok := withdrawalPathID(w, r)
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_review", "审核参数无效")
		return
	}
	request.Reason = strings.TrimSpace(request.Reason)
	if len(request.Reason) < 3 || len(request.Reason) > 500 {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_review", "审核原因长度无效")
		return
	}
	var withdrawal MerchantWithdrawal
	var err error
	if action == "approve" {
		withdrawal, err = s.services.Settlement.ApproveMerchantWithdrawal(r.Context(), id, session.UserID, request.Reason, time.Now().UTC())
	} else {
		withdrawal, err = s.services.Settlement.RejectMerchantWithdrawal(r.Context(), id, session.UserID, request.Reason, time.Now().UTC())
	}
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": withdrawal})
}

func (s *server) adminPayWithdrawal(w http.ResponseWriter, r *http.Request) {
	_, session, ok := s.requireSession(w, r, roleAdmin)
	if !ok || !s.requireSameOrigin(w, r) || !s.settlementReady(w) {
		return
	}
	id, ok := withdrawalPathID(w, r)
	if !ok {
		return
	}
	var request struct {
		PayoutReference string `json:"payout_reference"`
		PayoutProof     string `json:"payout_proof"`
	}
	if err := decodeJSON(r, &request); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_payment", "打款凭证参数无效")
		return
	}
	request.PayoutReference = strings.TrimSpace(request.PayoutReference)
	request.PayoutProof = strings.TrimSpace(request.PayoutProof)
	if len(request.PayoutReference) < 3 || len(request.PayoutReference) > 200 || len(request.PayoutProof) < 3 || len(request.PayoutProof) > 2000 {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_payment", "打款编号或凭证说明长度无效")
		return
	}
	withdrawal, err := s.services.Settlement.PayMerchantWithdrawal(
		r.Context(), id, session.UserID, request.PayoutReference, request.PayoutProof, time.Now().UTC(),
	)
	if err != nil {
		s.writeSettlementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": withdrawal})
}

func (s *server) settlementReady(w http.ResponseWriter) bool {
	if s.services.Settlement == nil || s.services.Credentials == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "settlement_unavailable", "结算服务尚未就绪")
		return false
	}
	return true
}

func (s *server) writeSettlementError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrSettlementNotFound):
		writeAPIError(w, http.StatusNotFound, "settlement_not_found", "结算记录不存在")
	case errors.Is(err, ErrInsufficientMerchantBalance):
		writeAPIError(w, http.StatusConflict, "insufficient_merchant_balance", "可提现余额不足")
	case errors.Is(err, ErrSettlementConflict):
		writeAPIError(w, http.StatusConflict, "settlement_state_conflict", "结算状态已变化，请刷新后重试")
	default:
		writeAPIError(w, http.StatusInternalServerError, "settlement_failed", "结算操作失败")
	}
}

func withdrawalPathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_withdrawal_id", "提现申请编号无效")
		return 0, false
	}
	return id, true
}

func newWithdrawalNo(now time.Time) (string, error) {
	var random [5]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "MW" + now.Format("20060102150405") + strings.ToUpper(hex.EncodeToString(random[:])), nil
}

func maskRecipientAccount(account string) string {
	characters := []rune(strings.TrimSpace(account))
	if len(characters) <= 7 {
		return strings.Repeat("*", len(characters))
	}
	return string(characters[:3]) + "****" + string(characters[len(characters)-4:])
}
