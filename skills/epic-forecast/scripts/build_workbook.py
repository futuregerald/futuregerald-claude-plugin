#!/usr/bin/env python3
"""Build a styled .xlsx forecast workbook from a JSON manifest.

Usage:
    python3 build_workbook.py manifest.json out.xlsx

Requires openpyxl. If it is missing, create a venv in the scratchpad:
    python3 -m venv venv && ./venv/bin/pip install openpyxl
    ./venv/bin/python build_workbook.py manifest.json out.xlsx

Manifest schema
---------------
{
  "title":    "Platform Delivery — Work Status & Forecast",
  "subtitle": "Verified against the tracker and GitHub on 2026-01-31.",
  "sheets": [
    {
      "name":    "Summary",
      "note":    "Optional line rendered under the title.",
      "columns": [
        {"header": "Item", "key": "item", "width": 38, "wrap": true},
        {"header": "Status", "key": "status", "width": 16, "style": "status"},
        {"header": "Opt (wks)", "key": "opt", "width": 10, "align": "center"},
        {"header": "Link", "key": "url", "width": 30, "style": "link"}
      ],
      "rows": [ {"item": "...", "status": "In Progress", "opt": 2.0, "url": "https://..."} ],
      "group_by": "group"
    }
  ]
}

Column styles: "status" (colour-codes by value), "link" (hyperlink; pair with `text` key
`<key>_text` for display text), "risk" (red/amber/green by value), "num" (1 decimal).
Row-level overrides: a row may carry "_fill" (hex, no #) to tint the whole row.
"group_by" inserts a bold banner row whenever that field's value changes.
"""

import json
import sys

from openpyxl import Workbook
from openpyxl.styles import Alignment, Border, Font, PatternFill, Side
from openpyxl.utils import get_column_letter

INK = "1F2933"
MUTED = "5C6B7A"
HEADER_BG = "1F3A5F"
BANNER_BG = "DCE6F1"
STRIPE = "F5F8FB"

STATUS_FILLS = {
    "done": ("C6EFCE", "0B6B32"),
    "in progress": ("FFF0C2", "8A6100"),
    "in development": ("FFF0C2", "8A6100"),
    "in review": ("FFF0C2", "8A6100"),
    "planning": ("E3E0F5", "4A3F8F"),
    "planning & design": ("E3E0F5", "4A3F8F"),
    "research": ("E3E0F5", "4A3F8F"),
    "backlog": ("EDF0F3", "5C6B7A"),
    "to do": ("EDF0F3", "5C6B7A"),
    "not started": ("EDF0F3", "5C6B7A"),
    "no ticket": ("FBD5D5", "9B1C1C"),
    "won't do": ("E8E8E8", "8A8A8A"),
    "blocked": ("FBD5D5", "9B1C1C"),
}

RISK_FILLS = {
    "high": ("FBD5D5", "9B1C1C"),
    "critical": ("FBD5D5", "9B1C1C"),
    "medium": ("FFF0C2", "8A6100"),
    "med": ("FFF0C2", "8A6100"),
    "low": ("C6EFCE", "0B6B32"),
}

THIN = Side(style="thin", color="D5DCE4")
BORDER = Border(left=THIN, right=THIN, top=THIN, bottom=THIN)


def _lookup(table, value):
    if value is None:
        return None
    key = str(value).strip().lower()
    if key in table:
        return table[key]
    for name, colours in table.items():
        if key.startswith(name):
            return colours
    return None


def _write_title(ws, title, subtitle, note, ncols):
    ws.cell(row=1, column=1, value=title).font = Font(bold=True, size=15, color=INK)
    ws.merge_cells(start_row=1, start_column=1, end_row=1, end_column=max(ncols, 1))
    ws.row_dimensions[1].height = 24

    row = 2
    for text in (subtitle, note):
        if not text:
            continue
        ws.cell(row=row, column=1, value=text).font = Font(size=10, color=MUTED, italic=True)
        ws.merge_cells(start_row=row, start_column=1, end_row=row, end_column=max(ncols, 1))
        row += 1
    return row + 1


def build_sheet(wb, spec, subtitle, first):
    ws = wb.active if first else wb.create_sheet()
    ws.title = spec["name"][:31]

    columns = spec["columns"]
    header_row = _write_title(ws, spec.get("title", spec["name"]), subtitle, spec.get("note"), len(columns))

    for idx, col in enumerate(columns, start=1):
        cell = ws.cell(row=header_row, column=idx, value=col["header"])
        cell.font = Font(bold=True, color="FFFFFF", size=10)
        cell.fill = PatternFill("solid", fgColor=HEADER_BG)
        cell.alignment = Alignment(vertical="center", wrap_text=True)
        cell.border = BORDER
        ws.column_dimensions[get_column_letter(idx)].width = col.get("width", 18)
    ws.row_dimensions[header_row].height = 30

    group_key = spec.get("group_by")
    current_group = object()
    r = header_row + 1
    stripe = False

    for entry in spec["rows"]:
        if group_key and entry.get(group_key) != current_group:
            current_group = entry.get(group_key)
            if current_group:
                cell = ws.cell(row=r, column=1, value=str(current_group))
                cell.font = Font(bold=True, size=10, color=INK)
                cell.fill = PatternFill("solid", fgColor=BANNER_BG)
                ws.merge_cells(start_row=r, start_column=1, end_row=r, end_column=len(columns))
                for c in range(1, len(columns) + 1):
                    ws.cell(row=r, column=c).fill = PatternFill("solid", fgColor=BANNER_BG)
                r += 1
                stripe = False

        row_fill = entry.get("_fill")
        for idx, col in enumerate(columns, start=1):
            value = entry.get(col["key"])
            cell = ws.cell(row=r, column=idx)
            style = col.get("style")

            if style == "link" and value:
                cell.value = entry.get(col["key"] + "_text") or value
                cell.hyperlink = value
                cell.font = Font(color="1155CC", underline="single", size=10)
            else:
                cell.value = value
                cell.font = Font(size=10, color=INK)

            if style == "num" and isinstance(value, (int, float)):
                cell.number_format = "0.0"
            elif style == "status":
                colours = _lookup(STATUS_FILLS, value)
                if colours:
                    cell.fill = PatternFill("solid", fgColor=colours[0])
                    cell.font = Font(size=10, bold=True, color=colours[1])
            elif style == "risk":
                colours = _lookup(RISK_FILLS, value)
                if colours:
                    cell.fill = PatternFill("solid", fgColor=colours[0])
                    cell.font = Font(size=10, bold=True, color=colours[1])

            already_tinted = style in ("status", "risk") and cell.fill.patternType == "solid"
            if row_fill and not already_tinted:
                cell.fill = PatternFill("solid", fgColor=row_fill)
            elif stripe and not already_tinted and not row_fill:
                cell.fill = PatternFill("solid", fgColor=STRIPE)

            cell.alignment = Alignment(
                vertical="top",
                wrap_text=bool(col.get("wrap")),
                horizontal=col.get("align", "left"),
            )
            cell.border = BORDER
        r += 1
        stripe = not stripe

    ws.freeze_panes = ws.cell(row=header_row + 1, column=1)
    ws.auto_filter.ref = f"A{header_row}:{get_column_letter(len(columns))}{r - 1}"
    ws.sheet_view.showGridLines = False
    return ws


def main():
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(2)

    with open(sys.argv[1]) as fh:
        manifest = json.load(fh)

    wb = Workbook()
    for i, spec in enumerate(manifest["sheets"]):
        build_sheet(wb, spec, manifest.get("subtitle", ""), first=(i == 0))

    wb.save(sys.argv[2])
    print(f"wrote {sys.argv[2]} with {len(manifest['sheets'])} sheets")


if __name__ == "__main__":
    main()
