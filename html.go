package main

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RenderHTML writes a standalone HTML report (no external assets).
func RenderHTML(out Output, verify bool, ignoreExtras bool, filename string) error {
	type pageData struct {
		Out          Output
		Verify       bool
		IgnoreExtras bool
		Generated    string
	}
	data := pageData{
		Out:          out,
		Verify:       verify,
		IgnoreExtras: ignoreExtras,
		Generated:    time.Now().Format("2006-01-02 15:04:05 MST"),
	}

	funcs := template.FuncMap{
		"comma":      func(i any) string { return CommaAny(i) },
		"bytesHuman": BytesHuman,
		"itemVersionCount": func(it Item) int {
			return len(it.Versions)
		},
		"itemGhostCount": func(it Item, verify bool) int {
			if !verify {
				return 0
			}
			n := 0
			for _, v := range it.Versions {
				for _, p := range v.Parts {
					if !p.VerifiedOnDisk {
						n++
					}
				}
			}
			return n
		},
		"safeID": func(s string) string {
			s = strings.ToLower(s)
			repl := []string{" ", "-", "/", "-", "\\", "-", ".", "-", ":", "-", "#", "-", "?", "-", "&", "-"}
			for i := 0; i+1 < len(repl); i += 2 {
				s = strings.ReplaceAll(s, repl[i], repl[i+1])
			}
			return s
		},
		"policyName": func(s string) string {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "ignore-4k-1080":
				return "Policy: Ignore 4K+1080 pair"
			default:
				return "Policy: Plex (all multi-version)"
			}
		},
	}

	const tpl = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>PLEX Super Duper Report</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
:root {
  --bg:#0f172a;--panel:#111827;--muted:#94a3b8;--text:#e5e7eb;--accent:#38bdf8;
  --ok:#10b981;--warn:#f59e0b;--bad:#ef4444;--chip:#1f2937;--border:#1f2937;
}
*{box-sizing:border-box}
body{margin:0;font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,Cantarell,Noto Sans,Helvetica,Arial,Apple Color Emoji,Segoe UI Emoji;background:var(--bg);color:var(--text)}
.container{max-width:1200px;margin:0 auto;padding:24px}
h1{font-size:28px;margin:0 0 8px}
h2{font-size:22px;margin-top:32px}
h3{font-size:18px;margin:18px 0 8px}
.panel{background:var(--panel);border:1px solid var(--border);border-radius:12px;padding:16px}
.grid{display:grid;gap:12px}
.grid-2{grid-template-columns:repeat(2,minmax(0,1fr))}
.kv{display:grid;grid-template-columns:1fr auto;gap:6px}
.muted{color:var(--muted)}
.chips{display:flex;flex-wrap:wrap;gap:8px}
.chip{padding:2px 8px;background:var(--chip);border:1px solid var(--border);border-radius:999px;font-size:12px}
.chip.ok{border-color:var(--ok);color:#d1fae5}
.chip.bad{border-color:var(--bad);color:#fee2e2}
.chip.warn{border-color:var(--warn);color:#fff7ed}
a{color:var(--accent);text-decoration:none}
a:hover{text-decoration:underline}
table{width:100%;border-collapse:collapse;margin-top:8px}
th,td{text-align:left;padding:6px 8px;border-bottom:1px solid var(--border);vertical-align:top}
code{background:#0b1220;padding:2px 4px;border-radius:6px}
.summary-cards{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-top:12px}
.card{background:var(--panel);border:1px solid var(--border);border-radius:12px;padding:14px}
.card h3{margin:0 0 6px;font-size:16px}
.small{font-size:12px}
.details{margin-top:12px}
details{border:1px solid var(--border);border-radius:10px;padding:10px 12px;background:#0b1325}
details+details{margin-top:8px}
details summary{cursor:pointer;font-weight:600}
.badge{font-size:11px;padding:2px 6px;border-radius:8px;background:var(--chip);border:1px solid var(--border);margin-left:6px}
.badge.ok{border-color:var(--ok);color:#22c55e}
.badge.bad{border-color:var(--bad);color:#ef4444}
.badge.warn{border-color:var(--warn);color:#f59e0b}
.toc a{display:inline-block;margin-right:12px;margin-bottom:8px}
.footer{margin-top:24px;color:var(--muted);font-size:12px}
hr{border:none;height:1px;background:var(--border);margin:20px 0}
@media (max-width: 960px) {
  .summary-cards{grid-template-columns:repeat(2,minmax(0,1fr))}
}
</style>
</head>
<body>
<div class="container">
  <header>
    <h1>PLEX Super Duper Report</h1>
    <div class="muted small">Generated: {{ .Generated }} &nbsp;•&nbsp; Server: <code>{{ .Out.Server }}</code></div>
    <div class="chips" style="margin-top:8px">
      {{ if .Verify }}
        <span class="chip ok">Verification: On (checkFiles)</span>
      {{ else }}
        <span class="chip warn">Verification: Off (ghost counts not checked)</span>
      {{ end }}
      <span class="chip">{{ policyName .Out.Summary.DuplicatePolicy }}</span>
     {{ if gt (len .Out.Ignored) -1 }}
      <span class="chip">{{ if .IgnoreExtras }}Extras: Ignored{{ else }}Extras: Included{{ end }}</span>
     {{ end }}
      {{/* Extras flag chip (we don’t have it in Summary, so derive from data):
            if any section/item exists we can’t tell from JSON alone; simplest is
            to pass Options.IgnoreExtras into RenderHTML; if you prefer, modify
            RenderHTML signature to take ignoreExtras bool as a second flag. */}}
    </div>
  </header>

  <section class="panel" style="margin-top:16px">
    <h2>Summary</h2>
    <div class="summary-cards">
      <div class="card"><h3>Total Duplicates</h3><div style="font-size:26px;font-weight:700">{{ comma .Out.Summary.TotalDuplicateItems }}</div></div>
      <div class="card"><h3>Total Libraries</h3><div style="font-size:26px;font-weight:700">{{ comma .Out.Summary.TotalLibraries }}</div></div>
      <div class="card"><h3>Total Versions</h3><div style="font-size:26px;font-weight:700">{{ comma .Out.TotalVersions }}</div></div>
      <div class="card"><h3>Total Ghost Parts</h3><div style="font-size:26px;font-weight:700">{{ comma .Out.Summary.TotalGhostParts }}</div></div>
      {{ if gt .Out.Summary.VariantItemsExcluded 0 }}
      <div class="card"><h3>4K+1080 Pairs Ignored</h3><div style="font-size:26px;font-weight:700">{{ comma .Out.Summary.VariantItemsExcluded }}</div></div>
      {{ end }}
    </div>

    <h3 style="margin-top:16px">Per-Library</h3>
    <div class="toc">
      {{ range .Out.Summary.Libraries }}<a href="#lib-{{ safeID .SectionID }}">{{ .SectionTitle }}</a>{{ end }}
    </div>
    <div class="grid">
      {{ range .Out.Summary.Libraries }}
      <div class="panel">
        <h3 id="lib-{{ safeID .SectionID }}">{{ .SectionTitle }}</h3>
        <div class="grid grid-2">
          <div class="kv"><span>Duplicate items</span><strong>{{ comma .DuplicateItems }}</strong></div>
          <div class="kv"><span>Total versions</span><strong>{{ comma .TotalVersions }}</strong></div>
          <div class="kv"><span>Items with ghosts</span><strong>{{ comma .ItemsWithGhosts }}</strong></div>
          <div class="kv"><span>Ghost parts</span><strong>{{ comma .GhostParts }}</strong></div>
          {{ if gt .VariantsExcluded 0 }}
          <div class="kv"><span>4K+1080 pairs ignored</span><strong>{{ comma .VariantsExcluded }}</strong></div>
          {{ end }}
        </div>
      </div>
      {{ end }}
    </div>
  </section>

  <section class="details">
    <h2>Details (All Duplicate Items)</h2>
    {{ range $s := .Out.Sections }}
      <h3 style="margin-top:18px">{{ $s.SectionTitle }} <span class="badge">{{ len $s.Items }} items</span></h3>
      {{ if eq (len $s.Items) 0 }}
        <div class="muted">No duplicates in this library after applying policy.</div>
      {{ else }}
        {{ range $it := $s.Items }}
        <details>
          <summary>
            {{ $it.Title }}{{ if $it.Year }} ({{ $it.Year }}){{ end }}
            <span class="badge">{{ itemVersionCount $it }} versions</span>
            {{ $gc := itemGhostCount $it $.Verify }}
            {{ if $.Verify }}
              {{ if gt $gc 0 }}<span class="badge bad">{{ $gc }} ghost{{ if gt $gc 1 }}s{{ end }}</span>{{ else }}<span class="badge ok">no ghosts</span>{{ end }}
            {{ else }}<span class="badge warn">verification off</span>{{ end }}
          </summary>
          <table>
            <thead><tr><th>Version</th><th>Codec</th><th>Resolution</th><th>Part File</th><th>Size</th><th>Status</th></tr></thead>
            <tbody>
              {{ range $v := $it.Versions }}
                {{ range $p := $v.Parts }}
                <tr>
                  <td><code>{{ $v.Container }}</code></td>
                  <td><span class="muted">{{ $v.VideoCodec }}</span> / <span class="muted">{{ $v.AudioCodec }}</span></td>
                  <td>{{ $v.VideoResolution }} ({{ $v.Width }}×{{ $v.Height }})</td>
                  <td><code>{{ $p.File }}</code></td>
                  <td>{{ bytesHuman $p.Size }}</td>
                  <td>
                    {{ if $.Verify }}
                      {{ if $p.VerifiedOnDisk }}<span class="chip ok">Verified</span>{{ else }}<span class="chip bad">Missing/Unreachable</span>{{ end }}
                    {{ else }}<span class="chip warn">Not checked</span>{{ end }}
                  </td>
                </tr>
                {{ end }}
              {{ end }}
            </tbody>
          </table>
        </details>
        {{ end }}
      {{ end }}
    {{ end }}
  </section>

  {{ if gt (len .Out.Ignored) 0 }}
  <section class="details" style="margin-top:22px">
    <h2>Ignored (4K+1080 Pairs)</h2>
    <div class="muted small" style="margin-bottom:8px">
      The items below were not counted as duplicates because they contain exactly one 4K (≈2160p) and one 1080p version, with no other versions.
    </div>
    {{ range $ig := .Out.Ignored }}
    <details>
      <summary>
        {{ $ig.Item.Title }}{{ if $ig.Item.Year }} ({{ $ig.Item.Year }}){{ end }}
        <span class="badge">{{ $ig.SectionTitle }}</span>
        <span class="badge ok">Reason: 4K+1080 pair</span>
      </summary>
      <table>
        <thead><tr><th>Version</th><th>Codec</th><th>Resolution</th><th>Part File</th><th>Size</th><th>Status</th></tr></thead>
        <tbody>
          {{ range $v := $ig.Item.Versions }}
            {{ range $p := $v.Parts }}
            <tr>
              <td><code>{{ $v.Container }}</code></td>
              <td><span class="muted">{{ $v.VideoCodec }}</span> / <span class="muted">{{ $v.AudioCodec }}</span></td>
              <td>{{ $v.VideoResolution }} ({{ $v.Width }}×{{ $v.Height }})</td>
              <td><code>{{ $p.File }}</code></td>
              <td>{{ bytesHuman $p.Size }}</td>
              <td>
                {{ if $.Verify }}
                  {{ if $p.VerifiedOnDisk }}<span class="chip ok">Verified</span>{{ else }}<span class="chip bad">Missing/Unreachable</span>{{ end }}
                {{ else }}<span class="chip warn">Not checked</span>{{ end }}
              </td>
            </tr>
            {{ end }}
          {{ end }}
        </tbody>
      </table>
    </details>
    {{ end }}
  </section>
  {{ end }}

  <div class="footer">Report generated by <strong>goPlexr</strong>. Self-contained file.</div>
</div>
</body>
</html>`

	t, err := template.New("report").Funcs(funcs).Parse(tpl)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(filename); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return t.Execute(f, data)
}
