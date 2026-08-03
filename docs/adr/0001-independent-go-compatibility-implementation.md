# Use an independent Go compatibility implementation

OrangeCount will be an independent Go implementation that reads and evaluates existing Beancount syntax and semantics without a Python Beancount runtime. Although wrapping the reference implementation would reduce initial effort, independent execution is necessary to address its diagnostics, performance, and maintainability limitations while retaining the user's current ledger.
