import { Check, Download, Eye, Search, ShieldCheck, X } from 'lucide-react';
import { merchantReviews } from '../../data';
import { Button, Section, Status, TextInput } from '../../ui';

export default function AdminMerchants({ notify }) {
  return (
    <Section
      title="商户申请审核"
      description="管理员验证网址、上游密钥和最小真实请求后，才开通同账号商户身份"
      actions={<Button icon={Download}>导出申请</Button>}
    >
      <div className="filter-toolbar slim-toolbar"><label className="search-box"><Search size={16} /><TextInput placeholder="搜索申请人、网址或编号" /></label><Button variant="primary">筛选</Button></div>
      <div className="table-wrap"><table className="data-table"><thead><tr><th>申请编号</th><th>申请人</th><th>网站</th><th>拟售厂商</th><th>提交时间</th><th>状态</th><th>审核操作</th></tr></thead><tbody>
        {merchantReviews.map((review) => <tr key={review.id}><td><code>{review.id}</code></td><td><strong>{review.applicant}</strong><small>UID {review.id.slice(-4)}</small></td><td>{review.website}</td><td>{review.provider}</td><td>{review.submitted}</td><td><Status>{review.state}</Status></td><td><div className="row-actions"><Button icon={Eye} variant="ghost">详情</Button><Button icon={ShieldCheck} variant="ghost" onClick={() => notify(`已发起 ${review.id} 渠道验证`)}>验证</Button><Button icon={Check} variant="ghost" onClick={() => notify(`${review.id} 已通过审核`)}>通过</Button><Button icon={X} variant="danger" onClick={() => notify(`${review.id} 已驳回`)}>驳回</Button></div></td></tr>)}
      </tbody></table></div>
    </Section>
  );
}
