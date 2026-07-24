import { ShieldCheck } from 'lucide-react';
import { Button, Modal, Status } from './ui';

const matchNames = { matched: '已匹配', unmatched: '平台未收录' };
const testNames = { passed: '检查通过', failed: '检查失败', untested: '未检查', pending: '未检查' };

function channelModelItems(item) {
  if (item?.kind === 'review') return item.models || [];
  const names = item?.matched_models?.length ? item.matched_models : (item?.model ? [item.model] : []);
  return names.map((name, index) => ({ id: `${item.id}-${index}`, upstream_model_name: name, match_status: 'imported' }));
}

export function ChannelModelsButton({ item, onClick }) {
  const count = channelModelItems(item).length;
  return <Button variant="ghost" onClick={onClick}>查看模型（{count}）</Button>;
}

export default function ChannelModelsModal({ item, busyID = '', onClose, onTest }) {
  const models = channelModelItems(item);
  const isReview = item?.kind === 'review';
  return (
    <Modal title="渠道模型" description={`渠道 ID ${item.id} · ${item.channel_name}`} onClose={onClose} wide>
      <div className="table-wrap channel-models-wrap">
        <table className="data-table channel-models-table">
          <thead><tr><th>上游模型</th><th>匹配状态</th><th>检查状态</th><th>耗时</th><th>错误</th>{onTest && <th>操作</th>}</tr></thead>
          <tbody>
            {models.map((model) => <tr key={model.id || model.upstream_model_name}>
              <td><code>{model.upstream_model_name}</code></td>
              <td><Status>{isReview ? (matchNames[model.match_status] || model.match_status) : '已导入'}</Status></td>
              <td>{isReview ? <Status>{testNames[model.test_status] || model.test_status || '未检查'}</Status> : '-'}</td>
              <td>{model.test_latency_ms ? `${model.test_latency_ms}ms` : '-'}</td>
              <td className="channel-model-error">{model.test_error || '-'}</td>
              {onTest && <td>{isReview && model.match_status === 'matched'
                ? <Button disabled={Boolean(busyID)} icon={ShieldCheck} variant="ghost" onClick={() => onTest(item, model)}>{busyID === `model-${model.id}` ? '检查中' : 'hi 检查'}</Button>
                : '-'}</td>}
            </tr>)}
            {models.length === 0 && <tr><td className="table-message" colSpan={onTest ? 6 : 5}>未获取到模型</td></tr>}
          </tbody>
        </table>
      </div>
    </Modal>
  );
}
