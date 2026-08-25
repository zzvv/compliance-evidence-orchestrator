package transport

import "net/http"

const consolePage = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>合规证据编排</title><style>body{margin:0;background:#f4f6f5;color:#1c2723;font:15px system-ui,sans-serif}header{background:#163c32;color:white;padding:20px 5vw}main{max-width:960px;margin:28px auto;padding:0 22px}.grid{display:grid;grid-template-columns:repeat(3,1fr);gap:14px}.panel{background:#fff;border:1px solid #d7dfda;padding:18px;border-radius:6px}.value{font-size:28px;color:#16654c;margin-top:8px}h1{font-size:22px;margin:0}h2{font-size:17px;margin-top:28px}table{width:100%;border-collapse:collapse;background:white}th,td{text-align:left;padding:12px;border-bottom:1px solid #e4e9e6}th{font-weight:600;color:#50615a}</style></head><body><header><h1>合规证据编排中心</h1></header><main><section class="grid"><div class="panel">待审核<div class="value">0</div></div><div class="panel">本周完成<div class="value">0</div></div><div class="panel">通知失败<div class="value">0</div></div></section><h2>最近审核批次</h2><table><thead><tr><th>项目</th><th>材料</th><th>状态</th><th>更新时间</th></tr></thead><tbody><tr><td colspan="4">当前没有审核批次</td></tr></tbody></table></main></body></html>`

func (h *Handler) console(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(consolePage))
}
