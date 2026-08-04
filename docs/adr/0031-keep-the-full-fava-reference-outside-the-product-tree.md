# Keep the full Fava reference outside the product tree

OrangeCount will maintain a read-only, repository-external Fava 1.30.12 checkout pinned to a recorded source commit for complete study and file-level provenance mapping. The project will commit only the revision lock, license evidence, mapping, and selected attributed frontend derivatives—not Fava's whole Python application, tests, or dependency tree—so parity work remains reproducible without inheriting unrelated runtime maintenance.
