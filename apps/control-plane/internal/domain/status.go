package domain

type EnvironmentStatus string

const (
	EnvironmentStatusPending      EnvironmentStatus = "PENDING"
	EnvironmentStatusProvisioning EnvironmentStatus = "PROVISIONING"
	EnvironmentStatusReady        EnvironmentStatus = "READY"
	EnvironmentStatusFailed       EnvironmentStatus = "FAILED"
	EnvironmentStatusDestroying   EnvironmentStatus = "DESTROYING"
	EnvironmentStatusDestroyed    EnvironmentStatus = "DESTROYED"
)

type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "PENDING"
	WorkflowStatusRunning   WorkflowStatus = "RUNNING"
	WorkflowStatusSucceeded WorkflowStatus = "SUCCEEDED"
	WorkflowStatusFailed    WorkflowStatus = "FAILED"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusRunning   StepStatus = "RUNNING"
	StepStatusSucceeded StepStatus = "SUCCEEDED"
	StepStatusFailed    StepStatus = "FAILED"
	StepStatusSkipped   StepStatus = "SKIPPED"
)

type Operation string

const (
	OperationCreate  Operation = "CREATE"
	OperationDestroy Operation = "DESTROY"
	OperationRetry   Operation = "RETRY"
)
