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

// VM drivers.
const (
	DriverEC2     = "ec2"
	DriverGCE     = "gce"
	DriverAzureVM = "azure-vm" // scaffold — Unimplemented until ARM Compute is wired
)

// Container drivers.
const (
	DriverDocker  = "docker"
	DriverECS     = "ecs" // reserved — use fargate adapter for AWS ECS/Fargate
	DriverGKE     = "gke" // reserved — use k8s adapter with a GKE kubeconfig
	DriverAKS     = "aks" // reserved — use k8s adapter with an AKS kubeconfig
	DriverK8s     = "k8s"
	DriverFargate = "fargate"
)

// Serverless drivers.
const (
	DriverLambda         = "lambda"
	DriverGCF            = "gcf"
	DriverAzureFunctions = "azure-functions" // Invoke via HTTP trigger URL; ARM CRUD scaffold
)
