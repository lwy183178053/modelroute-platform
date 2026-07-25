import { useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { Button, Modal, Pagination, Status } from './ui';

const healthNames = {
  pending: '待检查',
  healthy: '正常',
  cooling: '冷却',
  error: '错误',
};

function formatTime(value) {
  if (!value) return '-';
  return new Date(value).toLocaleString('zh-CN');
}

function formatNextCheck(value) {
  if (!value) return '-';
  const milliseconds = new Date(value).getTime() - Date.now();
  if (milliseconds <= 0) return '等待检查';
  const minutes = Math.max(1, Math.ceil(milliseconds / 60_000));
  if (minutes < 60) return `${minutes}分钟后`;
  return `${Math.ceil(minutes / 60)}小时后`;
}

function formatStatusCode(model) {
  const statusCode = Number(model?.last_status_code || 0);
  return Number.isInteger(statusCode) && statusCode >= 400 ? String(statusCode) : '-';
}

function channelModelItems(item) {
  return item?.models || [];
}

export function ChannelModelsButton({ item, manage = false, onClick }) {
  const count = channelModelItems(item).length;
  return <Button variant="ghost" onClick={onClick}>{manage ? '管理' : '查看'}模型（{count}）</Button>;
}

export default function ChannelModelsModal({ item, busyID = '', onClose, onTest }) {
  const models = channelModelItems(item);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const pageCount = Math.max(1, Math.ceil(models.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const visibleModels = models.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  useEffect(() => {
    setPage(1);
  }, [item?.id]);

  return (
    <Modal title="渠道模型" description={`渠道 ID ${item.id} · ${item.channel_name}`} onClose={onClose} wide className="channel-models-modal">
      <div className="table-wrap channel-models-wrap">
        <table className="data-table channel-models-table">
          <thead><tr>{onTest && <th className="channel-model-action-column">检查</th>}<th>具体模型</th><th>状态</th><th>真实失败</th><th>自动检查失败</th><th>下次检查</th><th>最后检查</th><th>耗时</th><th>错误码</th></tr></thead>
          <tbody>
            {visibleModels.map((model) => <tr key={model.id}>
              {onTest && <td className="channel-model-action-column"><Button disabled={Boolean(busyID)} icon={RefreshCw} variant="ghost" onClick={() => onTest(item, model)}>{busyID === `model-${model.id}` ? '检查中' : '检查'}</Button></td>}
              <td><code>{model.model || model.upstream_model_name}</code><small>{model.model && model.model !== model.upstream_model_name ? model.upstream_model_name : ''}</small></td>
              <td><Status>{healthNames[model.health_state] || model.health_state || '待检查'}</Status></td>
              <td>{Number(model.consecutive_request_failures || 0)}</td>
              <td>{Number(model.automatic_check_failures || 0)} / 10</td>
              <td>{model.health_state === 'cooling' ? formatNextCheck(model.next_check_at) : '-'}</td>
              <td>{formatTime(model.last_checked_at)}</td>
              <td>{model.last_check_latency_ms ? `${model.last_check_latency_ms}ms` : '-'}</td>
              <td className="channel-model-error" title={model.last_check_error || undefined}>{formatStatusCode(model)}</td>
            </tr>)}
            {models.length === 0 && <tr><td className="table-message" colSpan={onTest ? 9 : 8}>暂无已匹配模型，请先同步模型</td></tr>}
          </tbody>
        </table>
      </div>
      <Pagination
        page={currentPage}
        pageSize={pageSize}
        total={models.length}
        onPageChange={setPage}
        onPageSizeChange={(nextPageSize) => { setPageSize(nextPageSize); setPage(1); }}
        disabled={Boolean(busyID)}
      />
    </Modal>
  );
}
