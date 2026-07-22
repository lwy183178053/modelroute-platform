import { Check, Download, FileCheck2, ReceiptText, X } from 'lucide-react';
import { withdrawals } from '../../data';
import { Button, InlineNotice, Section, Status } from '../../ui';

export default function AdminWithdrawals({ notify }) {
  return (
    <Section title="人工提现审核" description="平台只冻结和审核提现金额，确认线下打款凭证后再完成出账" actions={<Button icon={Download}>导出申请</Button>}>
      <InlineNotice tone="warning">提现不自动打款。审核、打款确认和驳回均要求管理员原因并写入审计日志。</InlineNotice>
      <div className="table-wrap"><table className="data-table"><thead><tr><th>申请编号</th><th>商户</th><th>提现金额</th><th>申请时可用余额</th><th>收款账户</th><th>提交时间</th><th>状态</th><th>操作</th></tr></thead><tbody>
        {withdrawals.map((item) => <tr key={item.id}><td><code>{item.id}</code></td><td><strong>{item.merchant}</strong></td><td><strong>{item.amount}</strong></td><td>{item.available}</td><td>{item.account}</td><td>{item.submitted}</td><td><Status>{item.state}</Status></td><td><div className="row-actions"><Button icon={ReceiptText} variant="ghost">账本核对</Button><Button icon={Check} variant="ghost" onClick={() => notify(`${item.id} 审核通过，等待录入打款凭证`)}>通过</Button><Button icon={FileCheck2} variant="ghost" onClick={() => notify(`${item.id} 已确认打款`)}>确认打款</Button><Button icon={X} variant="danger">驳回</Button></div></td></tr>)}
      </tbody></table></div>
    </Section>
  );
}
