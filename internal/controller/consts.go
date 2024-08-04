package controller

const (
	annotatorKeyPrefix             = "annotator.ingress.kubernetes.io/"
	annotatorEnabledKey            = annotatorKeyPrefix + "enabled"
	annotatorLastAppliedRulesKey   = annotatorKeyPrefix + "last-applied-rules"
	annotatorLastAppliedVersionKey = annotatorKeyPrefix + "last-applied-version"
	annotatorReconcileNeededKey    = annotatorKeyPrefix + "reconcile-needed"
	annotatorRulesKey              = annotatorKeyPrefix + "rules"
)
