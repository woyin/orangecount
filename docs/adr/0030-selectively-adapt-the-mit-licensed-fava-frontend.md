# Selectively adapt the MIT-licensed Fava frontend

ADR-0023 is superseded: OrangeCount may selectively adapt Fava 1.30.12 frontend code, styles, and assets under Fava's MIT license, retaining the required copyright and license notices and recording every derived unit in the third-party notice inventory. Fava's Python/Beancount runtime, public HTTP API, plugin execution, and user extensions are not adopted; Go adapters connect the selected frontend behavior to OrangeCount's own semantic core, which gives the desired UX fidelity without turning the product into a whole-application fork.
