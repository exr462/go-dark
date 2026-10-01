package kbd

type InputField int

const (
	GitWorkspace InputField = iota
	GitUsername
	GitEmail
	GitMaxTagListSize
	JdkName
	JdkPath
	MvnName
	MvnPath
	FuzzyKey
	GitOperationsKey
	ProfileKey
	SessionKey
	DockerKey
	MvnKey
	JdkKey
	BuildKey
	QuitKey
	EditKey
	NewProjectKey
	SubmitKey
	CancelKey
	HelpKey
	ToggleKey
	EditShortcutsKey
	EditContent
	Terminal
	DeployKey
	// Ceiling ⚠ This must be always as last and not used in the inputs[id] ⚠
	Ceiling
)
