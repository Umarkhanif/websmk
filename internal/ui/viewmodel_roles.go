package ui

type LetterView struct {
	ID           string
	LetterNumber string
	LetterType   string
	StudentName  string
	Recipient    string
	IssueDate    string
	SignerName   string
	SignerRole   string
	HmacHash     string
}

type LedgerEntryView struct {
	ID               int64
	InvoiceID        string
	ReceiptNumber    string
	TxType           string
	AmountDelta      int
	StudentName      string
	ActorName        string
	PinVerified      bool
	ReversalApproval string
	SignatureHmac    string
	CreatedAt        string
}

type AssignmentView struct {
	ID               string
	ClassName        string
	SubjectName      string
	Title            string
	Description      string
	DueDate          string
	TeacherName      string
	Submitted        bool
	SubmissionID     string
	SubmissionStatus string
	Score            int
	SubmissionCount  int
}

type PklPlacementView struct {
	ID              string
	CompanyName     string
	IndustrySector  string
	Address         string
	GpsLat          float64
	GpsLng          float64
	RadiusMeters    int
	SupervisorName  string
	SupervisorPhone string
	SupervisorPin   string
}

type PklLogView struct {
	ID             string
	StudentID      string
	StudentName    string
	CompanyName    string
	Timestamp      string
	DistanceMeters int
	GeofenceStatus string
	ExifVerified   bool
}

type PklJournalView struct {
	ID                 string
	StudentID          string
	StudentName        string
	PlacementID        string
	CompanyName        string
	Date               string
	ActivityTitle      string
	CompetencyLearned  string
	SupervisorApproved bool
	TeacherApproved    bool
}

type StudentDocView struct {
	ID           string
	StudentID    string
	StudentName  string
	DocType      string
	DocName      string
	FileName     string
	BlobID       string
	FileSizeKB   int
	UploadedAt   string
	Status       string
	RejectReason string
}

type TaskSubmissionView struct {
	ID           string
	AssignmentID string
	StudentID    string
	StudentName  string
	SubjectName  string
	TaskTitle    string
	FileName     string
	BlobID       string
	FileSizeKB   int
	SubmittedAt  string
	Status       string
	Score        int
}

type BkkJobView struct {
	ID             string
	CompanyName    string
	JobTitle       string
	TargetedMajors string
	SalaryRange    string
	DeadlineDate   string
	ApplicantCount int
	AlreadyApplied bool
}

type BkkAppView struct {
	ID          string
	JobID       string
	JobTitle    string
	CompanyName string
	StudentID   string
	StudentName string
	AppliedDate string
	Status      string
}

type StudentExpenseView struct {
	ID          string
	StudentID   string
	Category    string
	ItemTitle   string
	Amount      int
	Date        string
	ReceiptNote string
}

type ExamQuestionView struct {
	ID             string
	QuestionNumber int
	QuestionType   string
	SubjectName    string
	KdCode         string
	QuestionText   string
	Options        []QuizOptionView
	CorrectKey     string
	EssayRubric    string
}

type ExamScheduleView struct {
	ID            string
	SubjectName   string
	TeacherName   string
	DateLabel     string
	TimeSlot      string
	DurationMin   int
	QuestionCount int
	Room          string
	SessionNumber int
	Status        string // "READY", "SUBMITTED", "WAITING", "UPCOMING"
	StatusBadge   string
	BadgeClass    string
	Score         int
	ActionURL     string
	Note          string
}

type ProctorSessionView struct {
	ID            string
	StudentID     string
	Name          string
	NISN          string
	Progress      string
	Status        string
	WarnCount     int
	LockCount     int
	LastViolation string
	UnlockToken   string
	Emergency2h   bool
}

type AttendanceRowView struct {
	ID             string
	StudentID      string
	StudentName    string
	NISN           string
	Status         string
	StatusLabel    string
	GateTapTime    string
	VerifiedSource string
}

type DispensationView struct {
	ID            string
	StudentID     string
	StudentName   string
	ExamPeriod    string
	ClearanceType string
	Reason        string
	ValidUntil    string
	GrantedBy     string
}

type ReversalRequestView struct {
	ID              string
	LedgerID        int64
	TxID            string
	CashierName     string
	AmountOriginal  int
	AmountCorrected int
	Reason          string
	Status          string
	OtpCode         string
}

type ConsultationView struct {
	ID          string
	StudentID   string
	StudentName string
	ParentName  string
	ChosenPath  string
	TopicNote   string
	Status      string
	CreatedAt   string
}

type ViolationView struct {
	ID               string
	StudentID        string
	StudentName      string
	ArticleCode      string
	ArticleTitle     string
	SourceType       string
	PointsDeducted   int
	Status           string
	SummonLetterHash string
	CreatedAt        string
}

type PeerReportView struct {
	ID                   string
	TargetStudentName    string
	TargetStudentClass   string
	ArticleCategory      string
	Location             string
	Description          string
	ReporterClusterHash  string
	DistinctClusterCount int
	Status               string
	CreatedAt            string
}

type CounselingView struct {
	ID                     string
	StudentID              string
	StudentName            string
	CounselorName          string
	IssueSummary           string
	StartDate              string
	EndDate                string
	RecoveryStatus         string
	DailyCheckinsCompleted int
	Notes                  string
}

type PiketClassroomView struct {
	ID         string
	Name       string
	Subject    string
	Teacher    string
	Time       string
	Room       string
	ScheduleID string
	Status     string
	Note       string
}

type SubstituteAssignmentView struct {
	ID              string
	ClassName       string
	Subject         string
	TeacherName     string
	OriginalTeacher string
	Note            string
}

type GateLogView struct {
	ID          string
	NISN        string
	StudentName string
	LogType     string
	Reason      string
	RecordedAt  string
	QrPassHash  string
}

type TeacherImpactView struct {
	ID                    string
	TeacherName           string
	NIP                   string
	Subject               string
	CalibratedScoreGrowth float64
	LowConfusionRate      float64
	LeaderValidationRate  float64
	SubstituteBonusPoints float64
	FinalImpactScore      float64
}

type MajorView struct {
	Code             string
	Name             string
	HeadOfMajor      string
	WorkshopRoom     string
	StudentCount     int
	DudiPartnerCount int
}

type DapodikLogView struct {
	ID            string
	SyncTimestamp string
	EntitySynced  string
	RecordsCount  int
	Status        string
}

type PublicVerifyPage struct {
	ActiveTab       string
	Query           string
	VerifiedLetter  *LetterView
	VerifiedReceipt *LedgerEntryView
	NotFound        bool
	SelectedJournal *PklJournalView
	Notice          string
	Error           string
	CSRFToken       string
}

type StudentPortalPage struct {
	Shell                  RoleShell
	Tab                    string
	Student                StudentView
	CbtScore               int
	AttendanceConductScore int
	HolisticScore          string
	Invoice                InvoiceView
	HasInvoice             bool
	IsCleared              bool
	HasDispensation        bool
	Journal                JournalView
	TeacherNotes           string
	AiSmartNotes           string
	Quiz                   QuizView
	QuizQuestions          []QuizQuestionView
	ReflectionLevel        string
	ReflectionComplaint    string
	ExamQuestions          []ExamQuestionView
	ExamSchedules          []ExamScheduleView
	ActiveExamID           string
	ActiveSubjectName      string
	ActiveSubjectTeacher   string
	ActiveSubjectRoom      string
	ActiveSubjectTime      string
	CompletedExamsCount    int
	TotalExamsCount        int
	ExamAverageScore       int
	AllExamsGraded         bool
	ExamTier               string
	Proctor                ProctorSessionView
	ExamSubmitted          bool
	ExamScore              int
	Documents              []StudentDocView
	Assignments            []AssignmentView
	TaskSubmissions        []TaskSubmissionView
	PklPlacement           PklPlacementView
	PklLogs                []PklLogView
	PklJournals            []PklJournalView
	BkkJobs                []BkkJobView
	BkkApps                []BkkAppView
	Expenses               []StudentExpenseView
	TotalExpense           int
}

type ParentLeaveView struct {
	ID        string
	LeaveType string
	LeaveDate string
	Note      string
	Status    string
	CreatedAt string
}

type ParentPortalPage struct {
	Shell                  RoleShell
	Tab                    string
	Student                StudentView
	CbtScore               int
	AttendanceConductScore int
	HolisticScore          string
	Attendance             AttendanceView
	ReflectionLevel        string
	ReflectionComplaint    string
	Invoices               []InvoiceView
	Ledger                 []LedgerEntryView
	Expenses               []StudentExpenseView
	TotalExpense           int
	Consultations          []ConsultationView
	Leaves                 []ParentLeaveView
}

type TreasurerPortalPage struct {
	Shell            RoleShell
	Tab              string
	TotalBilled      int
	TotalCollected   int
	TotalArrears     int
	CollectionRate   int
	Students         []StudentView
	Invoices         []InvoiceView
	Dispensations    []DispensationView
	Ledger           []LedgerEntryView
	ReversalRequests []ReversalRequestView
	ExpectedOTP      string
}

type ClassOptionView struct {
	ID   string
	Name string
}

type TeacherPortalPage struct {
	Shell            RoleShell
	Tab              string
	HomeroomClass    string
	ActiveClass      string
	SelectedClassID  string
	SelectedSchedule string
	SelectedSubject  string
	Classes          []ClassOptionView
	TodaySchedules   []ScheduleView
	ActiveStudent    StudentView
	CbtScore         int
	Journal          JournalView
	TeacherNotes     string
	AiSmartNotes     string
	Attendances      []AttendanceRowView
	ExamQuestions    []ExamQuestionView
	ProctorStudents  []ProctorSessionView
	ExamResetLogs    []ExamResetLogView
	PklLogs          []PklLogView
	PklJournals      []PklJournalView
	Assignments      []AssignmentView
	TaskSubmissions  []TaskSubmissionView
}

type DisciplinePortalPage struct {
	Shell               RoleShell
	Tab                 string
	Violations          []ViolationView
	PeerReports         []PeerReportView
	Consultations       []ConsultationView
	CounselingSessions  []CounselingView
	TransformationScore string
	TransformationTrend string
	RecoveredCount      int
	ActiveCount         int
}

type PiketPortalPage struct {
	Shell          RoleShell
	Classrooms     []PiketClassroomView
	TodaySchedules []ScheduleView
	Assignments    []SubstituteAssignmentView
	GateLogs       []GateLogView
	Teachers       []TeacherImpactView
}

type AccountView struct {
	ID          string
	Identifier  string
	FullName    string
	Role        string
	RoleLabel   string
	Phone       string
	Active      bool
	IsSelf      bool
	TOTPEnabled bool
	StudentName string
}

type AcademicYearView struct {
	ID        string
	Label     string
	Semester  string
	StartDate string
	EndDate   string
	Active    bool
}

type ClassroomView struct {
	ID           string
	Name         string
	GradeLevel   int
	MajorCode    string
	MajorName    string
	HomeroomID   string
	HomeroomName string
	StudentCount int
}

type ScheduleView struct {
	ID          string
	ClassroomID string
	ClassName   string
	SubjectID   string
	TeacherID   string
	TeacherName string
	DayName     string
	StartTime   string
	EndTime     string
	RoomName    string
}

type RolloverRowView struct {
	ClassID           string
	ClassName         string
	GradeLevel        int
	MajorCode         string
	StudentCount      int
	Targets           []ClassroomView
	SuggestedTargetID string
	Graduate          bool
}

type AdminTuPortalPage struct {
	Shell         RoleShell
	Tab           string
	Section       string
	Query         string
	Page          int
	PageCount     int
	StudentTotal  int
	StudentRows   []StudentDirectoryView
	Students      []StudentView
	Majors        []MajorView
	Teachers      []TeacherImpactView
	Documents     []StudentDocView
	Letters       []LetterView
	DapodikLogs   []DapodikLogView
	PklPlacements []PklPlacementView
	BkkJobs       []BkkJobView
	BkkApps       []BkkAppView
	Accounts      []AccountView
	AcademicYears []AcademicYearView
	Classrooms    []ClassroomView
	Schedules     []ScheduleView
	RolloverRows  []RolloverRowView
	RolloverYears []AcademicYearView
	RolloverReady bool
}

type StudentDirectoryView struct {
	ID            string
	NISN          string
	FullName      string
	ClassroomName string
	MajorCode     string
	ParentName    string
	ParentPhone   string
	IsAlumni      bool
}

type HeadmasterPortalPage struct {
	Shell               RoleShell
	Teachers            []TeacherImpactView
	ReversalRequests    []ReversalRequestView
	CounselingSessions  []CounselingView
	TransformationScore string
	TransformationTrend string
	ActiveOTP           string
	AiAdvice            string
	LastSignatureType   string
	LastSignatureHash   string
	LastSignatureAt     string
}

type BugReportView struct {
	ID             string
	ReporterUserID string
	ReporterName   string
	ReporterRole   string
	Category       string
	CategoryLabel  string
	Title          string
	Description    string
	PageURL        string
	Severity       string
	SeverityClass  string
	Status         string
	StatusClass    string
	DeveloperNotes string
	CreatedAt      string
}

type ExamResetLogView struct {
	ID          string
	StudentID   string
	StudentName string
	SubjectID   string
	ProctorName string
	Reason      string
	IsBulk      bool
	CreatedAt   string
}

type DeveloperPortalPage struct {
	Shell             RoleShell
	Tab               string
	TotalBugs         int
	OpenBugs          int
	InvestigatingBugs int
	ResolvedBugs      int
	BugReports        []BugReportView
	ExamResetLogs     []ExamResetLogView
}

type ContactPage struct {
	Shell         RoleShell
	ReporterName  string
	ReporterEmail string
	ReporterRole  string
	RefererURL    string
	SuccessMsg    string
	ErrorMsg      string
}
