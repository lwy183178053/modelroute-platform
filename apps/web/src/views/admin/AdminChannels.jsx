import { Ban, Download, HeartPulse, Play, RefreshCw, RotateCcw } from 'lucide-react';
import { merchantChannels } from '../../data';
import { Button, InlineNotice, Section, Status } from '../../ui';

export default function AdminChannels({ notify }) {
  return (
    <Section
      title="渠道与探活"
      description="按“渠道 + 模型”记录主动探活、真实请求结果、冷却与恢复"
      actions={<><Button icon={Download}>导出健康记录</Button><Button icon={RefreshCw} variant="primary" onClick={() => notify('全量探活任务已排队')}>全量探活</Button></>}
    >
      <InlineNotice tone="warning">冷却渠道不参与新请求路由；硬故障会同步停用对应 New API 渠道。</InlineNotice>
      <div className="table-wrap"><table className="data-table"><thead><tr><th>商户 / 渠道</th><th>绑定模型</th><th>15 分钟结果</th><th>连续失败</th><th>当前状态</th><th>状态截止</th><th>操作</th></tr></thead><tbody>
        {merchantChannels.map((channel, index) => <tr key={channel.channel}><td><strong>{index === 0 ? '北辰云智' : index === 1 ? '青川 API' : '星环中转'} · {channel.name}</strong><small>New API {channel.channel}</small></td><td>{index === 0 ? 'GPT-5.2' : index === 1 ? 'Claude Sonnet 4.5' : 'Gemini 3 Pro'}</td><td>{channel.availability} · {channel.latency}</td><td>{index === 2 ? 3 : 0}</td><td><Status>{channel.state}</Status></td><td>{index === 2 ? '10:26:00' : '-'}</td><td><div className="row-actions"><Button icon={Play} variant="ghost" onClick={() => notify('单次探活已排队')}>探活</Button><Button icon={RotateCcw} variant="ghost">解除冷却</Button><Button icon={Ban} variant="danger">停用</Button></div></td></tr>)}
      </tbody></table></div>
      <div className="health-events">
        <h3><HeartPulse size={17} />最近健康事件</h3>
        <div><Status>冷却</Status><strong>Gemini 备用线路 / Gemini 3 Pro</strong><span>连续 3 次上游超时，冷却 5 分钟</span><small>09:21:00</small></div>
        <div><Status>健康</Status><strong>OpenAI 主线路 / GPT-5.2</strong><span>主动探活成功，首字时间 1.18 秒</span><small>09:20:42</small></div>
      </div>
    </Section>
  );
}
