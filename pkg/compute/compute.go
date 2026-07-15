package compute

// Kind identifies a compute capability within the compute domain.
type Kind string

const (
	KindVM         Kind = "vm"
	KindContainer  Kind = "container"
	KindServerless Kind = "serverless"
)

// Shared / testing driver.
const (
	DriverMemory = "memory"
)

// VM drivers. Only memory ships today; cloud names are reserved placeholders.
const (
	DriverEC2     = "ec2"      // reserved — not implemented
	DriverGCE     = "gce"      // reserved — not implemented
	DriverAzureVM = "azure-vm" // reserved — not implemented
)

// Container drivers. Memory, k8s, and fargate ship; Docker/ECS/GKE/AKS names are reserved.
const (
	DriverDocker  = "docker" // reserved — not implemented
	DriverECS     = "ecs"    // reserved — use fargate adapter for AWS ECS/Fargate
	DriverGKE     = "gke"    // reserved — use k8s adapter with a GKE kubeconfig
	DriverAKS     = "aks"    // reserved — use k8s adapter with an AKS kubeconfig
	DriverK8s     = "k8s"
	DriverFargate = "fargate"
)

// Serverless drivers. Memory, lambda, and gcf ship; Azure Functions is reserved.
const (
	DriverLambda         = "lambda"
	DriverGCF            = "gcf"
	DriverAzureFunctions = "azure-functions" // reserved — not implemented
)
