package k8s

type NamespaceService struct {
	watchNS string
	userNS  []string
	all     []string
}

func NewNamespaceService(watchNS string, userNS []string) *NamespaceService {
	ns := &NamespaceService{
		watchNS: watchNS,
		userNS:  userNS,
	}
	all := []string{watchNS}
	all = append(all, userNS...)
	ns.all = all
	return ns
}

func (ns *NamespaceService) Namespaces() []string {
	return ns.all
}
