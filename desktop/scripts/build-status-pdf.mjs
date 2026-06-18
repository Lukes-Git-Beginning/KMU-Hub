// Build a nicely styled PDF from .planning/status-overview.md (incl. rendered Mermaid).
// Pipeline: markdown-it -> HTML (editorial CSS) -> Playwright/Chromium headless -> A4 PDF.
// Run: node desktop/scripts/build-status-pdf.mjs   (resolves deps from desktop/node_modules)

import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import MarkdownIt from 'markdown-it';
import { chromium } from 'playwright';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');
const buildDir = resolve(repoRoot, '.planning', 'pdf-build');
const mdPath = resolve(repoRoot, '.planning', 'status-overview.md');
const htmlPath = resolve(buildDir, 'snapshot.html');
const pdfPath = resolve(repoRoot, '.planning', 'status-overview.pdf');

if (!existsSync(mdPath)) throw new Error('Markdown nicht gefunden: ' + mdPath);
if (!existsSync(resolve(buildDir, 'mermaid.min.js'))) throw new Error('mermaid.min.js fehlt im build-Ordner');

// --- Markdown -> HTML, Mermaid-Fences als <pre class="mermaid"> ---
const md = new MarkdownIt({ html: true, linkify: true, typographer: true });
const defaultFence = md.renderer.rules.fence.bind(md.renderer.rules);
let diagramCount = 0;
md.renderer.rules.fence = (tokens, idx, options, env, self) => {
  const t = tokens[idx];
  if (t.info.trim() === 'mermaid') {
    diagramCount++;
    return `<pre class="mermaid">${md.utils.escapeHtml(t.content)}</pre>\n`;
  }
  return defaultFence(tokens, idx, options, env, self);
};

const markdown = readFileSync(mdPath, 'utf8');
const body = md.render(markdown);

const themeVars = {
  fontFamily: '"Plus Jakarta Sans", "Segoe UI", sans-serif',
  primaryColor: '#e6fffa'.replace(' ', ''),
  primaryTextColor: '#134e4a',
  primaryBorderColor: '#0f766e',
  lineColor: '#64748b',
  secondaryColor: '#f1f5f9',
  tertiaryColor: '#f8fafc',
  tertiaryTextColor: '#1e293b',
  pie1: '#0f766e', pie2: '#f59e0b', pie3: '#cbd5e1',
  pie4: '#0ea5e9', pie5: '#94a3b8', pie6: '#14b8a6',
  pieStrokeColor: '#ffffff', pieOuterStrokeColor: '#cbd5e1',
  pieTitleTextSize: '17px', pieSectionTextSize: '14px', pieLegendTextSize: '13px',
  doneTaskBkgColor: '#cbd5e1', doneTaskBorderColor: '#94a3b8',
  activeTaskBkgColor: '#0f766e', activeTaskBorderColor: '#0f766e',
  taskBkgColor: '#e2e8f0', taskBorderColor: '#94a3b8',
  taskTextColor: '#1e293b', taskTextDarkColor: '#1e293b', taskTextLightColor: '#ffffff',
  gridColor: '#e2e8f0', todayLineColor: '#dc2626',
  sectionBkgColor: '#f8fafc', altSectionBkgColor: '#ffffff', sectionBkgColor2: '#f1f5f9',
};

const html = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<title>Cosmi/Zentria CRM - Status-Snapshot</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=Playfair+Display:wght@600;700;800&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
  :root {
    --ink:#1e293b; --muted:#64748b; --faint:#94a3b8;
    --accent:#0f766e; --accent-soft:#e6fffa; --accent-line:#14b8a6;
    --border:#e2e8f0; --surface:#f8fafc; --surface-2:#f1f5f9;
  }
  * { box-sizing: border-box; }
  html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
  body {
    font-family:"Plus Jakarta Sans","Segoe UI",sans-serif;
    color:var(--ink); font-size:10.5pt; line-height:1.55;
    margin:0; padding:0;
  }
  /* --- Title block --- */
  h1 {
    font-family:"Playfair Display",Georgia,serif;
    font-size:27pt; line-height:1.12; font-weight:800; letter-spacing:-0.01em;
    color:#0f172a; margin:0 0 .35rem;
  }
  h1 + blockquote {
    margin-top:.4rem;
  }
  h2 {
    font-family:"Plus Jakarta Sans",sans-serif;
    font-size:15.5pt; font-weight:800; color:#0f172a; letter-spacing:-0.01em;
    margin:1.7rem 0 .7rem; padding-bottom:.35rem;
    border-bottom:2px solid var(--accent);
    break-after:avoid;
  }
  h3 { font-size:12pt; font-weight:700; color:#0f172a; margin:1.1rem 0 .4rem; break-after:avoid; }
  p { margin:.55rem 0; }
  strong { font-weight:700; color:#0f172a; }
  a { color:var(--accent); text-decoration:none; }
  hr { border:none; border-top:1px solid var(--border); margin:1.4rem 0; }
  ul,ol { margin:.5rem 0; padding-left:1.3rem; }
  li { margin:.2rem 0; }

  /* code */
  code {
    font-family:"JetBrains Mono",Consolas,monospace; font-size:8.8pt;
    background:var(--surface-2); color:#0f766e;
    padding:.07em .38em; border-radius:5px; border:1px solid var(--border);
  }

  /* blockquotes = the disclaimer / hint notes */
  blockquote {
    margin:.8rem 0; padding:.6rem .95rem; background:var(--surface);
    border-left:4px solid var(--accent-line); border-radius:0 8px 8px 0;
    color:var(--muted); font-size:9.6pt; break-inside:avoid;
  }
  blockquote p { margin:.25rem 0; }
  blockquote code { background:#fff; }

  /* tables */
  table {
    width:100%; border-collapse:collapse; margin:.9rem 0; font-size:9.2pt;
  }
  thead { display:table-header-group; }
  th {
    background:#0f172a; color:#fff; font-weight:600; text-align:left;
    padding:.42rem .6rem; border:1px solid #0f172a; font-size:8.8pt;
    letter-spacing:.01em;
  }
  td { padding:.38rem .6rem; border:1px solid var(--border); vertical-align:top; }
  tbody tr:nth-child(even) td { background:var(--surface); }
  tr { break-inside:avoid; }

  /* diagrams */
  pre.mermaid {
    margin:1rem 0; padding:1.1rem 1rem; background:#fff;
    border:1px solid var(--border); border-radius:12px;
    text-align:center; break-inside:avoid;
    box-shadow:0 1px 2px rgba(15,23,42,.04);
  }
  pre.mermaid svg { max-width:100%; height:auto; }

  /* captions: a <p> that contains only an <em> */
  p:has(> em:only-child) {
    margin:.2rem 0 1.2rem; padding-left:.9rem; border-left:3px solid var(--border);
    color:var(--muted); font-size:9.2pt; font-style:normal;
  }
  p:has(> em:only-child) em { font-style:italic; }
</style>
</head>
<body>
${body}
<script src="./mermaid.min.js"></script>
<script>
  window.__renderDone = false; window.__renderError = null;
  try {
    mermaid.initialize({
      startOnLoad:false, theme:'base', securityLevel:'loose',
      themeVariables:${JSON.stringify(themeVars)},
      flowchart:{ curve:'basis', htmlLabels:true, padding:14 },
      pie:{ useWidth:760 },
      gantt:{ barHeight:22, barGap:6, topPadding:42, leftPadding:160, fontSize:12, sectionFontSize:13, gridLineStartPadding:30 }
    });
    (async () => {
      try { await mermaid.run({ querySelector:'.mermaid' }); }
      catch (e) { window.__renderError = String(e && e.message || e); }
      window.__renderDone = true;
    })();
  } catch (e) { window.__renderError = String(e); window.__renderDone = true; }
</script>
</body>
</html>`;

writeFileSync(htmlPath, html, 'utf8');
console.log('HTML geschrieben:', htmlPath, '| Mermaid-Diagramme:', diagramCount);

// --- HTML -> PDF via Chromium ---
const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto(pathToFileURL(htmlPath).href, { waitUntil: 'networkidle', timeout: 60000 });
  await page.waitForFunction(() => window.__renderDone === true, { timeout: 60000 });
  const renderError = await page.evaluate(() => window.__renderError);
  const svgCount = await page.evaluate(() => document.querySelectorAll('pre.mermaid svg').length);
  if (renderError) console.warn('Mermaid-Warnung:', renderError);
  console.log('gerenderte SVGs:', svgCount, '/', diagramCount);
  await page.evaluate(() => document.fonts && document.fonts.ready);
  await page.emulateMedia({ media: 'print' });
  await page.pdf({
    path: pdfPath,
    format: 'A4',
    printBackground: true,
    margin: { top: '16mm', bottom: '16mm', left: '15mm', right: '15mm' },
    displayHeaderFooter: true,
    headerTemplate: '<div></div>',
    footerTemplate:
      '<div style="font-family:sans-serif; font-size:7.5pt; color:#94a3b8; width:100%; padding:0 15mm; display:flex; justify-content:space-between;">' +
      '<span>Cosmi / Zentria CRM &mdash; Projekt-Status-Snapshot (Stand 2026-06-18)</span>' +
      '<span>Seite <span class="pageNumber"></span> / <span class="totalPages"></span></span></div>',
  });
  console.log('PDF geschrieben:', pdfPath);
} finally {
  await browser.close();
}
