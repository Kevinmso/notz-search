"""Converte o relatório de entrega (Markdown) em PDF.

Uso:
    python3 -m venv .venv && .venv/bin/pip install weasyprint markdown-it-py pygments
    .venv/bin/python docs/build_pdf.py

Lê docs/lab-cicd-entrega.md e gera docs/lab-cicd-entrega.pdf.
Imagens (prints) podem ser inseridas no .md com ![legenda](evidencias/arquivo.png).
Trechos [PREENCHER ...] saem destacados em amarelo para não passarem despercebidos.
"""
import html
import re
import sys
from pathlib import Path

from markdown_it import MarkdownIt
from pygments import highlight
from pygments.formatters import HtmlFormatter
from pygments.lexers import TextLexer, get_lexer_by_name
from weasyprint import HTML

DOCS = Path(__file__).resolve().parent
SRC = Path(sys.argv[1]) if len(sys.argv) > 1 else DOCS / "lab-cicd-entrega.md"
OUT = Path(sys.argv[2]) if len(sys.argv) > 2 else SRC.with_suffix(".pdf")

formatter = HtmlFormatter(nowrap=True, style="default")


def render_fence(self, tokens, idx, options, env):
    token = tokens[idx]
    lang = token.info.strip().split()[0] if token.info.strip() else ""
    try:
        lexer = get_lexer_by_name(lang) if lang else TextLexer()
    except Exception:
        lexer = TextLexer()
    code = highlight(token.content, lexer, formatter)
    return f'<pre class="code"><code>{code}</code></pre>\n'


md = MarkdownIt("commonmark", {"html": False}).enable("table")
md.add_render_rule("fence", render_fence)

body = md.render(SRC.read_text(encoding="utf-8"))
body = re.sub(r"(\[PREENCHER.*?\])", r'<mark>\1</mark>', body, flags=re.S)

css = HtmlFormatter(style="default").get_style_defs(".code") + """
@page { size: A4; margin: 2cm 1.8cm;
  @bottom-right { content: counter(page) " / " counter(pages); font-size: 8pt; color: #666; } }
body { font-family: 'DejaVu Sans', sans-serif; font-size: 9.5pt; line-height: 1.45; color: #1a1a1a; }
h1 { font-size: 17pt; border-bottom: 2px solid #1b3a8a; padding-bottom: 4px; color: #1b3a8a; }
h2 { font-size: 13pt; color: #1b3a8a; margin-top: 18px; break-after: avoid; }
h3 { font-size: 10.5pt; margin-top: 14px; break-after: avoid; }
table { border-collapse: collapse; width: 100%; margin: 8px 0; font-size: 8.5pt; }
th, td { border: 1px solid #bbb; padding: 4px 6px; text-align: left; vertical-align: top; }
th { background: #eef1f8; }
code { font-family: 'DejaVu Sans Mono', monospace; font-size: 8pt; background: #f1f1f1; padding: 0 2px; }
pre.code { background: #f7f7f7; border: 1px solid #ddd; padding: 6px 8px; font-size: 7.3pt;
  line-height: 1.3; white-space: pre-wrap; overflow-wrap: anywhere; }
pre.code code { background: none; padding: 0; font-size: inherit; }
mark { background: #fff3a3; padding: 0 2px; }
img { max-width: 100%; border: 1px solid #ccc; margin: 6px 0; }
hr { border: 0; border-top: 1px solid #ccc; margin: 14px 0; }
li { margin: 2px 0; }
"""

doc = f"<!doctype html><html lang='pt-BR'><head><meta charset='utf-8'><style>{css}</style></head><body>{body}</body></html>"
HTML(string=doc, base_url=str(SRC.parent)).write_pdf(OUT)
print(f"PDF gerado: {OUT}")
