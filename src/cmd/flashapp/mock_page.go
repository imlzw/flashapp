package main

import (
	"bytes"
	"context"
	htmltmpl "html/template"
	"strconv"
	"strings"
	"time"
)

type mockMetric struct {
	Label string
	Value string
}

type mockFeature struct {
	Tag   string
	Title string
	Body  string
}

type mockPageData struct {
	Title          string
	Summary        string
	Prompt         string
	RevisionLabel  string
	Timestamp      string
	Accent         string
	AccentDark     string
	AccentSoft     string
	Surface        string
	SurfaceAlt     string
	Ink            string
	InkSoft        string
	Glow           string
	Layout         string
	LayoutLabel    string
	IsUpdate       bool
	Metrics        []mockMetric
	FeatureCards   []mockFeature
	TodoItems      []string
	ProductCards   []mockFeature
	Timeline       []mockFeature
	FocusPhrases   []string
	DeploymentHint string
}

var mockPageTemplate = htmltmpl.Must(htmltmpl.New("mock-page").Parse(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{ .Title }}</title>
  <style>
    :root {
      --accent: {{ .Accent }};
      --accent-dark: {{ .AccentDark }};
      --accent-soft: {{ .AccentSoft }};
      --surface: {{ .Surface }};
      --surface-alt: {{ .SurfaceAlt }};
      --ink: {{ .Ink }};
      --ink-soft: {{ .InkSoft }};
      --glow: {{ .Glow }};
    }
    * { box-sizing: border-box; }
    ::-webkit-scrollbar { width: 5px; height: 5px; }
    ::-webkit-scrollbar-track { background: transparent; }
    ::-webkit-scrollbar-thumb { background: rgba(0,0,0,0.1); border-radius: 10px; }
    @media (prefers-color-scheme: dark) {
      ::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); }
    }
    body {
      margin: 0;
      font-family: "Segoe UI Variable Display", "Noto Sans SC", "Microsoft YaHei UI", sans-serif;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(255,255,255,0.85), transparent 42%),
        linear-gradient(140deg, #fff4ea 0%, #fffdf9 48%, #f3fbff 100%);
      min-height: 100vh;
    }
    .orb {
      position: fixed;
      width: 22rem;
      height: 22rem;
      border-radius: 999px;
      filter: blur(10px);
      opacity: 0.22;
      pointer-events: none;
      background: var(--glow);
    }
    .orb-a { top: -8rem; left: -6rem; }
    .orb-b { right: -8rem; bottom: -6rem; }
    .page {
      position: relative;
      z-index: 1;
      max-width: 1160px;
      margin: 0 auto;
      padding: 32px 20px 48px;
    }
    .hero {
      background: linear-gradient(135deg, rgba(255,255,255,0.94), rgba(255,255,255,0.74));
      border: 1px solid rgba(20, 19, 18, 0.08);
      border-radius: 32px;
      padding: 28px;
      box-shadow: 0 30px 80px rgba(20, 19, 18, 0.08);
      backdrop-filter: blur(14px);
    }
    .eyebrow {
      display: inline-flex;
      gap: 10px;
      align-items: center;
      padding: 8px 12px;
      border-radius: 999px;
      background: rgba(255,255,255,0.86);
      border: 1px solid rgba(20, 19, 18, 0.08);
      font-size: 13px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--accent-dark);
    }
    .hero-grid {
      display: grid;
      gap: 24px;
      grid-template-columns: minmax(0, 1.25fr) minmax(300px, 0.75fr);
      margin-top: 18px;
    }
    h1 {
      margin: 0;
      font-size: clamp(2.1rem, 5vw, 4.8rem);
      line-height: 0.96;
      letter-spacing: -0.05em;
    }
    .summary {
      margin: 18px 0 0;
      max-width: 38rem;
      font-size: 1.05rem;
      line-height: 1.75;
      color: var(--ink-soft);
    }
    .capsule-row {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-top: 20px;
    }
    .capsule {
      padding: 10px 14px;
      border-radius: 999px;
      background: var(--surface-alt);
      border: 1px solid rgba(20, 19, 18, 0.08);
      font-size: 14px;
      color: var(--ink-soft);
    }
    .panel {
      background: linear-gradient(160deg, rgba(255,255,255,0.94), rgba(255,255,255,0.8));
      border: 1px solid rgba(20, 19, 18, 0.08);
      border-radius: 24px;
      padding: 22px;
      box-shadow: 0 22px 60px rgba(20, 19, 18, 0.06);
    }
    .metrics {
      display: grid;
      gap: 14px;
    }
    .metric {
      display: flex;
      align-items: baseline;
      justify-content: space-between;
      gap: 12px;
      padding: 14px 0;
      border-bottom: 1px solid rgba(20, 19, 18, 0.08);
    }
    .metric:last-child { border-bottom: none; padding-bottom: 0; }
    .metric-label { color: var(--ink-soft); font-size: 14px; }
    .metric-value { font-size: 1.3rem; font-weight: 700; color: var(--accent-dark); }
    .section {
      margin-top: 24px;
      display: grid;
      gap: 20px;
      grid-template-columns: minmax(0, 1fr) minmax(320px, 0.9fr);
    }
    .section-title {
      margin: 0 0 12px;
      font-size: 1rem;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--accent-dark);
    }
    .cards {
      display: grid;
      gap: 14px;
    }
    .card {
      background: rgba(255,255,255,0.78);
      border: 1px solid rgba(20, 19, 18, 0.08);
      border-radius: 20px;
      padding: 18px;
      transition: transform 180ms ease, box-shadow 180ms ease;
    }
    .card:hover {
      transform: translateY(-2px);
      box-shadow: 0 18px 40px rgba(20, 19, 18, 0.08);
    }
    .card-tag {
      display: inline-block;
      padding: 5px 10px;
      border-radius: 999px;
      background: rgba(255,255,255,0.92);
      color: var(--accent-dark);
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      border: 1px solid rgba(20, 19, 18, 0.08);
    }
    .card h3 {
      margin: 14px 0 8px;
      font-size: 1.2rem;
    }
    .card p {
      margin: 0;
      line-height: 1.7;
      color: var(--ink-soft);
    }
    .todo-list {
      display: grid;
      gap: 12px;
      margin-top: 12px;
    }
    .todo-item {
      display: grid;
      grid-template-columns: auto 1fr auto;
      gap: 12px;
      align-items: center;
      padding: 14px 16px;
      border-radius: 18px;
      background: rgba(255,255,255,0.84);
      border: 1px solid rgba(20, 19, 18, 0.08);
    }
    .todo-item input {
      width: 18px;
      height: 18px;
      accent-color: var(--accent);
    }
    .todo-item span { line-height: 1.5; }
    .todo-item strong {
      color: var(--accent-dark);
      font-size: 12px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
    }
    .timeline {
      display: grid;
      gap: 14px;
      margin-top: 12px;
    }
    .timeline-item {
      position: relative;
      padding: 18px 18px 18px 24px;
      border-radius: 18px;
      background: rgba(255,255,255,0.84);
      border: 1px solid rgba(20, 19, 18, 0.08);
      overflow: hidden;
    }
    .timeline-item::before {
      content: "";
      position: absolute;
      inset: 0 auto 0 0;
      width: 6px;
      background: linear-gradient(180deg, var(--accent), var(--accent-dark));
    }
    .timeline-item h4 {
      margin: 0 0 6px;
      font-size: 1rem;
    }
    .timeline-item p {
      margin: 0;
      line-height: 1.7;
      color: var(--ink-soft);
    }
    .grid-cards {
      display: grid;
      gap: 14px;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      margin-top: 12px;
    }
    .product {
      padding: 18px;
      border-radius: 18px;
      background: rgba(255,255,255,0.84);
      border: 1px solid rgba(20, 19, 18, 0.08);
      min-height: 180px;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
    .product strong {
      font-size: 1.08rem;
      line-height: 1.4;
    }
    .product p {
      margin: 10px 0 0;
      color: var(--ink-soft);
      line-height: 1.6;
    }
    .product span {
      color: var(--accent-dark);
      font-weight: 700;
      margin-top: 14px;
    }
    .footer {
      margin-top: 24px;
      display: flex;
      justify-content: space-between;
      gap: 12px;
      flex-wrap: wrap;
      color: var(--ink-soft);
      font-size: 14px;
    }
    .button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 13px 18px;
      border-radius: 999px;
      background: linear-gradient(135deg, var(--accent), var(--accent-dark));
      color: #fff;
      text-decoration: none;
      font-weight: 700;
      border: none;
      cursor: pointer;
      box-shadow: 0 18px 36px rgba(20, 19, 18, 0.12);
    }
    .mini-note {
      margin-top: 18px;
      padding: 14px 16px;
      border-radius: 18px;
      background: rgba(255,255,255,0.72);
      border: 1px dashed rgba(20, 19, 18, 0.12);
      color: var(--ink-soft);
      line-height: 1.6;
    }
    @media (max-width: 920px) {
      .hero-grid, .section {
        grid-template-columns: 1fr;
      }
      .page { padding: 20px 14px 36px; }
      .hero { padding: 20px; border-radius: 26px; }
    }
  </style>
</head>
<body>
  <div class="orb orb-a"></div>
  <div class="orb orb-b"></div>
  <main class="page">
    <section class="hero">
      <div class="eyebrow">
        <span>FlashApp</span>
        <span>{{ .LayoutLabel }}</span>
        {{ if .IsUpdate }}<span>迭代更新</span>{{ else }}<span>首次生成</span>{{ end }}
      </div>
      <div class="hero-grid">
        <div>
          <h1>{{ .Title }}</h1>
          <p class="summary">{{ .Summary }}</p>
          <div class="capsule-row">
            {{ range .FocusPhrases }}
              <span class="capsule">{{ . }}</span>
            {{ end }}
          </div>
          <div class="mini-note">
            <strong>最新需求：</strong>{{ .Prompt }}<br>
            <strong>部署提示：</strong>{{ .DeploymentHint }}
          </div>
        </div>
        <aside class="panel">
          <h2 class="section-title">运行指标</h2>
          <div class="metrics">
            {{ range .Metrics }}
              <div class="metric">
                <span class="metric-label">{{ .Label }}</span>
                <span class="metric-value">{{ .Value }}</span>
              </div>
            {{ end }}
          </div>
          <div class="footer" style="margin-top:18px">
            <span>{{ .RevisionLabel }}</span>
            <span>{{ .Timestamp }}</span>
          </div>
        </aside>
      </div>
    </section>

    <section class="section">
      <div class="panel">
        <h2 class="section-title">核心模块</h2>
        <div class="cards">
          {{ range .FeatureCards }}
            <article class="card">
              <span class="card-tag">{{ .Tag }}</span>
              <h3>{{ .Title }}</h3>
              <p>{{ .Body }}</p>
            </article>
          {{ end }}
        </div>
      </div>

      <div class="panel">
        {{ if eq .Layout "todo" }}
          <h2 class="section-title">任务视图</h2>
          <div class="todo-list" id="todo-list">
            {{ range .TodoItems }}
              <label class="todo-item">
                <input type="checkbox">
                <span>{{ . }}</span>
                <strong>Pending</strong>
              </label>
            {{ end }}
          </div>
          <div class="mini-note">点击复选框即可模拟完成状态，适合快速验证交互节奏。</div>
        {{ else if eq .Layout "store" }}
          <h2 class="section-title">展示卡片</h2>
          <div class="grid-cards">
            {{ range .ProductCards }}
              <article class="product">
                <div>
                  <strong>{{ .Title }}</strong>
                  <p>{{ .Body }}</p>
                </div>
                <span>{{ .Tag }}</span>
              </article>
            {{ end }}
          </div>
          <div class="mini-note">这个版本强调信息展示、卖点排序和移动端可扫读性。</div>
        {{ else }}
          <h2 class="section-title">执行路径</h2>
          <div class="timeline">
            {{ range .Timeline }}
              <article class="timeline-item">
                <h4>{{ .Title }}</h4>
                <p>{{ .Body }}</p>
              </article>
            {{ end }}
          </div>
          <div class="mini-note">适合用于官网、活动页、介绍页或需要较强叙事节奏的 H5 页面。</div>
        {{ end }}
      </div>
    </section>

    <footer class="footer">
      <span>Generated by FlashApp mock agent</span>
      <a class="button" href="#" onclick="window.scrollTo({top:0,behavior:'smooth'});return false;">返回顶部</a>
    </footer>
  </main>

  <script>
    document.querySelectorAll('.todo-item input').forEach(function (checkbox) {
      checkbox.addEventListener('change', function () {
        var item = checkbox.closest('.todo-item');
        var badge = item.querySelector('strong');
        item.style.opacity = checkbox.checked ? '0.65' : '1';
        badge.textContent = checkbox.checked ? 'Done' : 'Pending';
      });
    });
  </script>
</body>
</html>
`))

func streamMockHTML(ctx context.Context, req agentRequest, writer *streamDeploymentWriter) error {
	data := buildMockPageData(req)
	var out bytes.Buffer
	if err := mockPageTemplate.Execute(&out, data); err != nil {
		return err
	}

	content := out.String()
	for start := 0; start < len(content); start += 640 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := start + 640
		if end > len(content) {
			end = len(content)
		}
		if err := writer.WriteChunk(content[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func buildMockPageData(req agentRequest) mockPageData {
	title := pickTitle(req.Title, req.Prompt, extractTitleFromHTML(req.ExistingHTML))
	layout := inferLayout(req.Prompt)
	phrases := promptFragments(req.Prompt, 5)
	if len(phrases) == 0 {
		phrases = []string{"响应式布局", "单文件部署", "快速验证"}
	}
	palette := paletteFor(req.AppID)

	metrics := []mockMetric{
		{Label: "需求片段", Value: strconv.Itoa(len(promptFragments(req.Prompt, 12)))},
		{Label: "页面类型", Value: layoutLabel(layout)},
		{Label: "部署方式", Value: "Atomic"},
	}

	featureCards := defaultFeatureCards(layout, req.Prompt)
	timeline := []mockFeature{
		{Title: "描述意图", Body: "把最新需求转成单页叙事结构，让核心目标、用户动作和视觉氛围同时对齐。"},
		{Title: "即时预览", Body: "生成中的 HTML 按块输出，右侧 iframe 可直接看到页面逐步成形。"},
		{Title: "一键发布", Body: "流结束后通过临时文件替换正式 index.html，部署过程对外访问保持稳定。"},
	}
	productCards := []mockFeature{
		{Tag: "01", Title: "核心卖点", Body: "把最重要的利益点放在首屏，并给出清晰的行动入口。"},
		{Tag: "02", Title: "视觉节奏", Body: "用分区卡片组织信息，适配移动端滚动浏览和快速扫读。"},
		{Tag: "03", Title: "转化提醒", Body: "每个区块都保留下一步动作提示，减少停顿和理解成本。"},
	}
	todoItems := promptFragments(req.Prompt, 4)
	if len(todoItems) == 0 {
		todoItems = []string{"梳理首屏信息层级", "补足关键交互模块", "检查移动端排版", "完成落盘部署"}
	}

	revisionLabel := "基于最新指令完成生成"
	if req.ExistingHTML != "" {
		revisionLabel = "已结合现有应用进行迭代"
	}

	return mockPageData{
		Title:          title,
		Summary:        summarizePrompt(req.Prompt),
		Prompt:         strings.TrimSpace(req.Prompt),
		RevisionLabel:  revisionLabel,
		Timestamp:      time.Now().Format("2006-01-02 15:04"),
		Accent:         palette.accent,
		AccentDark:     palette.accentDark,
		AccentSoft:     palette.accentSoft,
		Surface:        palette.surface,
		SurfaceAlt:     palette.surfaceAlt,
		Ink:            palette.ink,
		InkSoft:        palette.inkSoft,
		Glow:           palette.glow,
		Layout:         layout,
		LayoutLabel:    layoutLabel(layout),
		IsUpdate:       req.ExistingHTML != "",
		Metrics:        metrics,
		FeatureCards:   featureCards,
		TodoItems:      todoItems,
		ProductCards:   productCards,
		Timeline:       timeline,
		FocusPhrases:   phrases,
		DeploymentHint: "生成完成后会自动写入磁盘并映射到独立预览地址。",
	}
}

func inferLayout(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "todo"), strings.Contains(lower, "待办"), strings.Contains(lower, "任务"), strings.Contains(lower, "计划"):
		return "todo"
	case strings.Contains(lower, "商品"), strings.Contains(lower, "商城"), strings.Contains(lower, "店铺"), strings.Contains(lower, "菜单"), strings.Contains(lower, "产品"), strings.Contains(lower, "sale"):
		return "store"
	default:
		return "default"
	}
}

func layoutLabel(layout string) string {
	switch layout {
	case "todo":
		return "任务型"
	case "store":
		return "展示型"
	default:
		return "叙事型"
	}
}

func defaultFeatureCards(layout, prompt string) []mockFeature {
	fragments := promptFragments(prompt, 6)
	if len(fragments) == 0 {
		fragments = []string{"清晰首屏", "重点模块", "稳定部署"}
	}

	labels := []string{"Intent", "Flow", "Deploy", "Preview", "Style", "Scale"}
	items := make([]mockFeature, 0, 3)
	for i := 0; i < len(fragments) && i < 3; i++ {
		body := "围绕“" + fragments[i] + "”组织页面区块、层级和重点动作，保证用户第一眼就能知道该做什么。"
		switch layout {
		case "todo":
			body = "把“" + fragments[i] + "”转成可执行的任务节点，支持快速扫描与逐项完成。"
		case "store":
			body = "让“" + fragments[i] + "”成为卖点表达的一部分，在移动端首屏也能被快速理解。"
		}
		items = append(items, mockFeature{
			Tag:   labels[i],
			Title: fragments[i],
			Body:  body,
		})
	}
	for len(items) < 3 {
		items = append(items, mockFeature{
			Tag:   labels[len(items)],
			Title: "自动补全模块",
			Body:  "当需求描述不完整时，页面会补上必要的信息组织和交互容器，方便继续迭代。",
		})
	}
	return items
}

type palette struct {
	accent     string
	accentDark string
	accentSoft string
	surface    string
	surfaceAlt string
	ink        string
	inkSoft    string
	glow       string
}

func paletteFor(seed string) palette {
	options := []palette{
		{
			accent:     "#f97316",
			accentDark: "#9a3412",
			accentSoft: "#ffedd5",
			surface:    "#fffaf5",
			surfaceAlt: "#fff1e7",
			ink:        "#1c1917",
			inkSoft:    "#57534e",
			glow:       "rgba(249,115,22,0.32)",
		},
		{
			accent:     "#0891b2",
			accentDark: "#155e75",
			accentSoft: "#cffafe",
			surface:    "#f5fdff",
			surfaceAlt: "#e7f8fd",
			ink:        "#082f49",
			inkSoft:    "#0f4c5c",
			glow:       "rgba(8,145,178,0.28)",
		},
		{
			accent:     "#dc2626",
			accentDark: "#991b1b",
			accentSoft: "#fee2e2",
			surface:    "#fff8f8",
			surfaceAlt: "#ffeaea",
			ink:        "#1f1414",
			inkSoft:    "#6b3131",
			glow:       "rgba(220,38,38,0.22)",
		},
	}

	sum := 0
	for _, r := range seed {
		sum += int(r)
	}
	return options[sum%len(options)]
}
