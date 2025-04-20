package notifier

type EmailNotifier interface {
	Send(to, subject, body string) error
}
