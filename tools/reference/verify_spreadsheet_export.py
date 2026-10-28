#!/usr/bin/env python3
# Copyright 2026 OrangeCount contributors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""Verify that one query export carries the same columns and values in CSV,
XLSX, and ODS form.

The three files come from the OrangeCount query endpoint
(``/api/v1/query?q=...&format=csv|xlsx|ods``). The script compares the
Results sheet of each file cell by cell against the CSV, and checks the Query
sheet of both spreadsheets holds the query text.

Structural checks use only the standard library (``zipfile`` and
``xml.etree.ElementTree``). ``openpyxl`` and ``odfpy`` are optional: when they
are importable the script also reads the workbooks with them, which is a second
independent decoder and catches a package that is well-formed XML but invalid
as a spreadsheet. Neither is a project dependency.

Usage::

    uv run --offline --project tools/reference python \
        tools/reference/verify_spreadsheet_export.py --dir /tmp/exports \
        --query "SELECT account, balance FROM accounts ORDER BY account"
"""

from __future__ import annotations

import argparse
import csv
import sys
import zipfile
from pathlib import Path
from typing import TYPE_CHECKING
from xml.etree import ElementTree

if TYPE_CHECKING:  # pragma: no cover
    from collections.abc import Sequence

XLSX_NS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
ODS_TABLE_NS = "urn:oasis:names:tc:opendocument:xmlns:table:1.0"
ODS_TEXT_NS = "urn:oasis:names:tc:opendocument:xmlns:text:1.0"
ODS_OFFICE_NS = "urn:oasis:names:tc:opendocument:xmlns:office:1.0"

REQUIRED_XLSX_PARTS = (
    "[Content_Types].xml",
    "_rels/.rels",
    "xl/workbook.xml",
    "xl/_rels/workbook.xml.rels",
    "xl/styles.xml",
    "xl/worksheets/sheet1.xml",
    "xl/worksheets/sheet2.xml",
)
REQUIRED_ODS_PARTS = ("mimetype", "content.xml", "styles.xml", "META-INF/manifest.xml")


class Failure(Exception):
    """A verification step failed; the message states which invariant broke."""


def read_csv_rows(path: Path) -> list[list[str]]:
    with path.open(newline="", encoding="utf-8") as handle:
        return [list(row) for row in csv.reader(handle)]


def xlsx_sheet_rows(archive: zipfile.ZipFile, name: str) -> list[list[str]]:
    root = ElementTree.fromstring(archive.read(name))
    rows: list[list[str]] = []
    for row in root.iter(f"{{{XLSX_NS}}}row"):
        values: list[str] = []
        for cell in row.iter(f"{{{XLSX_NS}}}c"):
            text = ""
            inline = cell.find(f"{{{XLSX_NS}}}is/{{{XLSX_NS}}}t")
            if inline is not None:
                text = inline.text or ""
            else:
                value = cell.find(f"{{{XLSX_NS}}}v")
                if value is not None:
                    text = value.text or ""
            values.append(text)
        rows.append(values)
    return rows


def ods_table_rows(archive: zipfile.ZipFile) -> list[list[str]]:
    root = ElementTree.fromstring(archive.read("content.xml"))
    tables = list(root.iter(f"{{{ODS_TABLE_NS}}}table"))
    if not tables:
        msg = "content.xml declares no table"
        raise Failure(msg)
    rows: list[list[str]] = []
    for row in tables[0].iter(f"{{{ODS_TABLE_NS}}}table-row"):
        values: list[str] = []
        for cell in row.iter(f"{{{ODS_TABLE_NS}}}table-cell"):
            text = "".join(
                "".join(paragraph.itertext()) for paragraph in cell.iter(f"{{{ODS_TEXT_NS}}}p")
            )
            if not text:
                text = cell.get(f"{{{ODS_OFFICE_NS}}}value") or ""
            values.append(text)
        rows.append(values)
    return rows


def ods_table_names(archive: zipfile.ZipFile) -> list[str]:
    root = ElementTree.fromstring(archive.read("content.xml"))
    return [table.get(f"{{{ODS_TABLE_NS}}}name") or "" for table in root.iter(f"{{{ODS_TABLE_NS}}}table")]


def check_required_parts(archive: zipfile.ZipFile, names: Sequence[str], required: Sequence[str]) -> None:
    missing = [part for part in required if part not in names]
    if missing:
        msg = f"archive is missing required parts {missing}; found {sorted(names)}"
        raise Failure(msg)


def check_ods_mimetype_first(archive: zipfile.ZipFile) -> None:
    first = archive.infolist()[0]
    if first.filename != "mimetype":
        msg = f"first ODS entry is {first.filename!r}, must be 'mimetype'"
        raise Failure(msg)
    if first.compress_type != zipfile.ZIP_STORED:
        msg = "ODS mimetype entry must be stored uncompressed"
        raise Failure(msg)
    media_type = archive.read("mimetype").decode("ascii")
    if media_type != "application/vnd.oasis.opendocument.spreadsheet":
        msg = f"unexpected ODS media type {media_type!r}"
        raise Failure(msg)


def check_xml_well_formed(archive: zipfile.ZipFile, names: Sequence[str]) -> None:
    for name in names:
        if not name.endswith((".xml", ".rels")):
            continue
        try:
            ElementTree.fromstring(archive.read(name))
        except ElementTree.ParseError as error:
            msg = f"part {name} is not well-formed XML: {error}"
            raise Failure(msg) from error


def compare_tables(csv_rows: list[list[str]], xlsx_rows: list[list[str]], ods_rows: list[list[str]]) -> None:
    if not csv_rows:
        msg = "CSV export is empty"
        raise Failure(msg)
    for label, rows in (("xlsx", xlsx_rows), ("ods", ods_rows)):
        if len(rows) != len(csv_rows):
            msg = f"row count differs: csv={len(csv_rows)} {label}={len(rows)}"
            raise Failure(msg)
    for index, csv_row in enumerate(csv_rows):
        for column, want in enumerate(csv_row):
            for label, rows in (("xlsx", xlsx_rows), ("ods", ods_rows)):
                row = rows[index]
                got = row[column] if column < len(row) else None
                if got != want:
                    msg = f"row {index} column {column} {label}={got!r} csv={want!r}"
                    raise Failure(msg)


def check_optional_decoders(xlsx: Path, ods: Path) -> list[str]:
    notes: list[str] = []
    try:
        import openpyxl  # noqa: PLC0415
    except ImportError:
        notes.append("openpyxl not installed; XLSX checked structurally only")
    else:
        workbook = openpyxl.load_workbook(xlsx, read_only=True)
        if workbook.sheetnames != ["Results", "Query"]:
            msg = f"openpyxl sees sheets {workbook.sheetnames}"
            raise Failure(msg)
        notes.append(f"openpyxl accepted the workbook ({workbook.sheetnames})")
    try:
        from odf.opendocument import load  # noqa: PLC0415
    except ImportError:
        notes.append("odfpy not installed; ODS checked structurally only")
    else:
        document = load(str(ods))
        if not document.spreadsheet:
            msg = "odfpy sees no spreadsheet body"
            raise Failure(msg)
        notes.append("odfpy accepted the document")
    return notes


def verify(directory: Path, query: str | None) -> list[str]:
    csv_path = directory / "query_result.csv"
    xlsx_path = directory / "query_result.xlsx"
    ods_path = directory / "query_result.ods"
    for path in (csv_path, xlsx_path, ods_path):
        if not path.exists():
            msg = f"missing export {path}"
            raise Failure(msg)

    csv_rows = read_csv_rows(csv_path)
    notes: list[str] = []

    with zipfile.ZipFile(xlsx_path) as archive:
        names = archive.namelist()
        check_required_parts(archive, names, REQUIRED_XLSX_PARTS)
        check_xml_well_formed(archive, names)
        xlsx_rows = xlsx_sheet_rows(archive, "xl/worksheets/sheet1.xml")
        query_sheet = archive.read("xl/worksheets/sheet2.xml").decode("utf-8")
    notes.append(f"xlsx parts={len(names)}")

    with zipfile.ZipFile(ods_path) as archive:
        names = archive.namelist()
        check_required_parts(archive, names, REQUIRED_ODS_PARTS)
        check_xml_well_formed(archive, names)
        check_ods_mimetype_first(archive)
        ods_rows = ods_table_rows(archive)
        tables = ods_table_names(archive)
        content = archive.read("content.xml").decode("utf-8")
    if tables[:2] != ["Results", "Query"]:
        msg = f"ods sheet names are {tables}"
        raise Failure(msg)
    notes.append(f"ods parts={len(names)}")

    compare_tables(csv_rows, xlsx_rows, ods_rows)
    notes.append(f"all three formats carry {len(csv_rows)} rows x {len(csv_rows[0])} columns identically")

    if query:
        for label, body in (("xlsx", query_sheet), ("ods", content)):
            if query not in body:
                msg = f"{label} Query sheet does not contain the query text"
                raise Failure(msg)
        notes.append("both spreadsheets record the query text")

    notes.extend(check_optional_decoders(xlsx_path, ods_path))
    return notes


def main(argv: Sequence[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", type=Path, required=True, help="directory holding query_result.csv/.xlsx/.ods")
    parser.add_argument("--query", default=None, help="query text expected in the Query sheet")
    arguments = parser.parse_args(argv)
    try:
        notes = verify(arguments.dir, arguments.query)
    except Failure as error:
        print(f"FAIL: {error}")
        return 1
    for note in notes:
        print(f"ok: {note}")
    print("PASS: CSV, XLSX, and ODS carry the same columns and values")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
