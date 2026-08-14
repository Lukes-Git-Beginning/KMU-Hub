// Build a styled A4 PDF from any .planning/*.md document.
// Pipeline: markdown-it -> HTML (editorial CSS, self-hosted fonts) -> Playwright/Chromium -> PDF.
//
// Usage: node desktop/scripts/build-planning-pdf.mjs <markdown-path> [pdf-path]
//        node desktop/scripts/build-planning-pdf.mjs .planning/preis-und-kostenanalyse-2026-08-13.md
//
// Fonts are inlined as base64 data URIs, so the build works offline and the PDF
// never depends on a CDN — same reason `desktop/src/renderer/src/styles/fonts.css`
// self-hosts them. Mermaid is rendered only if the document contains diagrams
// and `.planning/pdf-build/mermaid.min.js` is present.

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'node:fs';
import { dirname, resolve, basename } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import MarkdownIt from 'markdown-it';
import { chromium } from 'playwright';

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(__dirname, '..', '..');

const mdArg = process.argv[2];
if (!mdArg) {
  console.error('Aufruf: node desktop/scripts/build-planning-pdf.mjs <markdown-path> [pdf-path]');
  process.exit(1);
}
const mdPath = resolve(repoRoot, mdArg);
if (!existsSync(mdPath)) throw new Error('Markdown nicht gefunden: ' + mdPath);

const pdfPath = process.argv[3]
  ? resolve(repoRoot, process.argv[3])
  : mdPath.replace(/\.md$/, '.pdf');

const buildDir = resolve(repoRoot, '.planning', 'pdf-build');
mkdirSync(buildDir, { recursive: true });
const htmlPath = resolve(buildDir, basename(mdPath).replace(/\.md$/, '.html'));

// --- Fonts: inline as base64 so the document is self-contained ---
const fontDir = resolve(repoRoot, 'desktop', 'src', 'renderer', 'public', 'fonts');
const dataUri = (file) => {
  const p = resolve(fontDir, file);
  if (!existsSync(p)) throw new Error('Font fehlt: ' + p);
  return `data:font/woff2;base64,${readFileSync(p).toString('base64')}`;
};
const LATIN = 'U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+2000-206F,U+2074,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD';
const LATIN_EXT = 'U+0100-024F,U+0259,U+1E00-1EFF,U+2020,U+20A0-20AB,U+20AD-20CF,U+2113,U+2C60-2C7F,U+A720-A7FF';

const fontFaces = `
@font-face{font-family:'Plus Jakarta Sans';font-style:normal;font-weight:400 800;src:url(${dataUri('plus-jakarta-sans-latin-ext.woff2')}) format('woff2');unicode-range:${LATIN_EXT};}
@font-face{font-family:'Plus Jakarta Sans';font-style:normal;font-weight:400 800;src:url(${dataUri('plus-jakarta-sans-latin.woff2')}) format('woff2');unicode-range:${LATIN};}
@font-face{font-family:'Plus Jakarta Sans';font-style:italic;font-weight:400 500;src:url(${dataUri('plus-jakarta-sans-italic-latin-ext.woff2')}) format('woff2');unicode-range:${LATIN_EXT};}
@font-face{font-family:'Plus Jakarta Sans';font-style:italic;font-weight:400 500;src:url(${dataUri('plus-jakarta-sans-italic-latin.woff2')}) format('woff2');unicode-range:${LATIN};}
@font-face{font-family:'Playfair Display';font-style:normal;font-weight:600 800;src:url(${dataUri('playfair-display-latin-ext.woff2')}) format('woff2');unicode-range:${LATIN_EXT};}
@font-face{font-family:'Playfair Display';font-style:normal;font-weight:600 800;src:url(${dataUri('playfair-display-latin.woff2')}) format('woff2');unicode-range:${LATIN};}
@font-face{font-family:'JetBrains Mono';font-style:normal;font-weight:400 500;src:url(${dataUri('jetbrains-mono-latin-ext.woff2')}) format('woff2');unicode-range:${LATIN_EXT};}
@font-face{font-family:'JetBrains Mono';font-style:normal;font-weight:400 500;src:url(${dataUri('jetbrains-mono-latin.woff2')}) format('woff2');unicode-range:${LATIN};}`;

// --- Markdown -> HTML ---
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

// Keep a number and its unit on the same line — in narrow table columns a plain
// space lets "1.250 €" break after the digits, which reads like a typo.
const NBSP = String.fromCharCode(0x00A0);
const typeset = (src) => src.replace(/(\d)[ ](€|%|GB|TB|PT|h\b|pp\b)/g, `$1${NBSP}$2`);

const body = md.render(typeset(markdown));

const docTitle = (markdown.match(/^#\s+(.+)$/m)?.[1] ?? basename(mdPath, '.md')).trim();
const docDate = markdown.match(/(\d{4}-\d{2}-\d{2})/)?.[1] ?? '';
const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

const hasMermaid = diagramCount > 0 && existsSync(resolve(buildDir, 'mermaid.min.js'));
if (diagramCount > 0 && !hasMermaid) {
  console.warn(`Warnung: ${diagramCount} Mermaid-Block/Blöcke, aber mermaid.min.js fehlt in ${buildDir} — werden als Text gesetzt.`);
}

const themeVars = {
  fontFamily: '"Plus Jakarta Sans", sans-serif',
  primaryColor: '#e6fffa', primaryTextColor: '#134e4a', primaryBorderColor: '#0f766e',
  lineColor: '#64748b', secondaryColor: '#f1f5f9', tertiaryColor: '#f8fafc',
  tertiaryTextColor: '#1e293b',
  pie1: '#0f766e', pie2: '#f59e0b', pie3: '#cbd5e1',
  pie4: '#0ea5e9', pie5: '#94a3b8', pie6: '#14b8a6',
  pieStrokeColor: '#ffffff', pieOuterStrokeColor: '#cbd5e1',
  pieTitleTextSize: '17px', pieSectionTextSize: '14px', pieLegendTextSize: '13px',
};

const html = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="utf-8">
<title>${esc(docTitle)}</title>
<style>
${fontFaces}
  :root {
    --ink:#1e293b; --deep:#0f172a; --muted:#64748b; --faint:#94a3b8;
    --accent:#0f766e; --accent-soft:#e6fffa; --accent-line:#14b8a6;
    --border:#e2e8f0; --surface:#f8fafc; --surface-2:#f1f5f9;
  }
  * { box-sizing:border-box; }
  html { -webkit-print-color-adjust:exact; print-color-adjust:exact; }
  body {
    font-family:"Plus Jakarta Sans","Segoe UI",sans-serif;
    color:var(--ink); font-size:10.4pt; line-height:1.55;
    margin:0; padding:0;
    font-variant-numeric:tabular-nums;
  }

  /* --- headings --- */
  h1 {
    font-family:"Playfair Display",Georgia,serif;
    font-size:28pt; line-height:1.1; font-weight:800; letter-spacing:-0.015em;
    color:var(--deep); margin:0 0 .5rem;
  }
  h1 + blockquote { margin-top:.6rem; }
  h2 {
    font-size:15.5pt; font-weight:800; color:var(--deep); letter-spacing:-0.012em;
    margin:1.9rem 0 .7rem; padding-bottom:.35rem;
    border-bottom:2px solid var(--accent);
    break-after:avoid; break-inside:avoid;
  }
  h3 {
    font-size:11.8pt; font-weight:700; color:var(--deep);
    margin:1.15rem 0 .4rem; break-after:avoid;
  }
  h4 { font-size:10.6pt; font-weight:700; color:var(--deep); margin:.9rem 0 .3rem; break-after:avoid; }
  p { margin:.55rem 0; }
  strong { font-weight:700; color:var(--deep); }
  a { color:var(--accent); text-decoration:none; word-break:break-word; }
  hr { border:none; border-top:1px solid var(--border); margin:1.5rem 0; }
  ul,ol { margin:.5rem 0; padding-left:1.3rem; }
  li { margin:.22rem 0; }

  /* --- code --- */
  code {
    font-family:"JetBrains Mono",Consolas,monospace; font-size:8.6pt;
    background:var(--surface-2); color:var(--accent);
    padding:.07em .38em; border-radius:5px; border:1px solid var(--border);
    word-break:break-word;
  }
  pre:not(.mermaid) {
    margin:1rem 0; padding:.9rem 1.1rem; background:var(--surface);
    border:1px solid var(--border); border-radius:10px;
    overflow-x:hidden; break-inside:avoid;
  }
  pre:not(.mermaid) code {
    background:none; border:none; padding:0; color:var(--ink);
    font-size:8.4pt; line-height:1.5; white-space:pre-wrap;
  }

  /* --- callouts --- */
  blockquote {
    margin:.85rem 0; padding:.65rem 1rem; background:var(--surface);
    border-left:4px solid var(--accent-line); border-radius:0 8px 8px 0;
    color:var(--muted); font-size:9.5pt; break-inside:avoid;
  }
  blockquote p { margin:.28rem 0; }
  blockquote strong { color:var(--deep); }
  blockquote code { background:#fff; }
  blockquote table { font-size:8.6pt; }

  /* --- tables --- */
  table { width:100%; border-collapse:collapse; margin:.9rem 0; font-size:9.1pt; }
  thead { display:table-header-group; }
  th {
    background:var(--deep); color:#fff; font-weight:600; text-align:left;
    padding:.42rem .6rem; border:1px solid var(--deep); font-size:8.7pt;
    letter-spacing:.01em;
  }
  /* inline markup inside a header must stay legible on the dark fill —
     without this, **bold** header cells render deep-on-deep, i.e. invisible */
  th strong, th em, th a { color:#fff; }
  th strong { font-weight:800; }
  th code {
    color:#fff; background:rgba(255,255,255,.14); border-color:rgba(255,255,255,.22);
  }
  td { padding:.38rem .6rem; border:1px solid var(--border); vertical-align:top; }
  tbody tr:nth-child(even) td { background:var(--surface); }
  tr { break-inside:avoid; }
  td strong { color:var(--deep); }

  /* --- diagrams --- */
  pre.mermaid {
    margin:1rem 0; padding:1.1rem 1rem; background:#fff;
    border:1px solid var(--border); border-radius:12px;
    text-align:center; break-inside:avoid;
  }
  pre.mermaid svg { max-width:100%; height:auto; }

  /* captions: a <p> holding only an <em> */
  p:has(> em:only-child) {
    margin:.2rem 0 1.2rem; padding-left:.9rem; border-left:3px solid var(--border);
    color:var(--muted); font-size:9.1pt; font-style:normal;
  }
  p:has(> em:only-child) em { font-style:italic; }
</style>
</head>
<body>
${body}
${hasMermaid ? `<script src="./mermaid.min.js"></script>
<script>
  window.__renderDone=false; window.__renderError=null;
  try {
    mermaid.initialize({ startOnLoad:false, theme:'base', securityLevel:'loose',
      themeVariables:${JSON.stringify(themeVars)},
      flowchart:{ curve:'basis', htmlLabels:true, padding:14 }, pie:{ useWidth:760 } });
    (async () => {
      try { await mermaid.run({ querySelector:'.mermaid' }); }
      catch (e) { window.__renderError = String(e && e.message || e); }
      window.__renderDone = true;
    })();
  } catch (e) { window.__renderError = String(e); window.__renderDone = true; }
</script>` : '<script>window.__renderDone=true;</script>'}
</body>
</html>`;

writeFileSync(htmlPath, html, 'utf8');
console.log('HTML:', htmlPath, hasMermaid ? `| Mermaid: ${diagramCount}` : '');

// --- HTML -> PDF ---
const browser = await chromium.launch();
try {
  const page = await browser.newPage();
  await page.goto(pathToFileURL(htmlPath).href, { waitUntil: 'networkidle', timeout: 60000 });
  await page.waitForFunction(() => window.__renderDone === true, { timeout: 60000 });
  const renderError = await page.evaluate(() => window.__renderError);
  if (renderError) console.warn('Mermaid-Warnung:', renderError);
  await page.evaluate(() => document.fonts && document.fonts.ready);
  await page.emulateMedia({ media: 'print' });

  const footerLeft = esc(docTitle) + (docDate ? ` &mdash; Stand ${docDate}` : '');
  await page.pdf({
    path: pdfPath,
    format: 'A4',
    printBackground: true,
    margin: { top: '15mm', bottom: '16mm', left: '15mm', right: '15mm' },
    displayHeaderFooter: true,
    headerTemplate: '<div></div>',
    footerTemplate:
      '<div style="font-family:sans-serif;font-size:7.5pt;color:#94a3b8;width:100%;padding:0 15mm;display:flex;justify-content:space-between;">' +
      `<span>${footerLeft}</span>` +
      '<span>Seite <span class="pageNumber"></span> / <span class="totalPages"></span></span></div>',
  });
  console.log('PDF:', pdfPath);
} finally {
  await browser.close();
}
