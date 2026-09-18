package network

// RemoveDevice forgets a cached node after any active scan finishes. A future
// discovery can find the device again; this does not modify the device itself.
func (s *Service) RemoveDevice(uid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	remaining := make(map[string]Device, len(s.known))
	var nodes []Device
	for address, node := range s.known {
		if node.UID != uid {
			remaining[address] = node
			nodes = append(nodes, node)
		}
	}
	if s.Store != nil {
		if err := s.Store.SaveDevices(UniqueDevices(nodes)); err != nil {
			return err
		}
	}
	s.known = remaining
	return nil
}
