package tunnel

//go:generate mockery --name=AgentRunner --output=./mocks

// AgentRunner abstracts launching and stopping the tunnel agent process
// (e.g. playit-cli binary).
type AgentRunner interface {
	Start(instanceName string, port int) (address string, err error)
	Stop(instanceName string) error
	Address(instanceName string) (string, bool)
}
