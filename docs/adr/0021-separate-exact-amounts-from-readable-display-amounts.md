# Separate exact amounts from readable display amounts

OrangeCount will retain exact rational values for accounting, validation, and machine-readable exports, but present amounts through a dedicated display layer. Non-terminating values created by lot booking will render as clearly approximate localized decimals with access to their exact value, rather than exposing implementation fractions such as `numerator/denominator` in normal reports.
