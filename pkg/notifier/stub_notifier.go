package notifier

type StubEmailNotifier struct{}

func NewStubEmailNotifier() *StubEmailNotifier {
	return &StubEmailNotifier{}
}

func (n *StubEmailNotifier) Send(to, subject, body string) error {
	return nil
}
