package tunnel

type TunnelService struct {
	runner AgentRunner
}

func NewTunnelService(runner AgentRunner) *TunnelService {
	return &TunnelService{runner: runner}
}

func (s *TunnelService) Start(instanceName string, port int) (string, error) {
	return s.runner.Start(instanceName, port)
}

func (s *TunnelService) Stop(instanceName string) error {
	return s.runner.Stop(instanceName)
}

func (s *TunnelService) Address(instanceName string) (string, bool) {
	return s.runner.Address(instanceName)
}
