package piazza

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/headzoo/surf/browser"
	"github.com/pkg/errors"
)

// LoginURL is the URL you need to login.
const LoginURL = `https://piazza.com/account/login`

// Client represents a client to the piazza API.
type Client struct {
	bow *browser.Browser
	aid string
}

// MakeClient returns a new logged in client.
func MakeClient(username, password string) (*Client, error) {
	c := &Client{
		bow: surf.NewBrowser(),
	}
	return c, c.Login(username, password)
}

// Login logs into Piazza with the specified username and password.
func (c *Client) Login(username, password string) error {
	if err := c.bow.Open(LoginURL); err != nil {
		return err
	}

	// Log in to the site.
	fm, err := c.bow.Form("form#login-form")
	if err != nil {
		return err
	}
	if err := fm.Input("email", username); err != nil {
		return err
	}
	if err := fm.Input("password", password); err != nil {
		return err
	}
	if err := fm.Submit(); err != nil {
		return err
	}
	code := c.bow.StatusCode()
	if code != 200 {
		return errors.Errorf("StatusCode = %d", code)
	}
	errText := c.bow.Dom().Find("#modal_error_text").Text()
	if len(errText) > 0 {
		return errors.Errorf(errText)
	}

	return nil
}

/*
   {
   	"content":"https://www.facebook.com/notes/facebook-engineering/the-full-stack-part-i/461505383919",
   	"subject":"Reading Sep 8: The Full Stack Part 1",
   	"created":"2016-09-06T20:32:57Z",
   	"id":"isrxno834nx6x2",
   	"config":{
   		"resource_type":"link",
   		"section":"general",
   		"date":""
   	}
   }
*/
type Resource struct {
	Content string `json:"content"`
	Subject string `json:"subject"`
	Created string `json:"created"`
	ID      string `json:"id"`
	Config  struct {
		ResourceType string `json:"resource_type"`
		Section      string `json:"section"`
		Date         string `json:"date"`
	} `json:"config"`
}

// FetchResources returns all the resources for a class.
func (c *Client) FetchResources(classResourceURL string) ([]Resource, error) {
	if err := c.bow.Open(classResourceURL); err != nil {
		return nil, err
	}
	body := ""
	c.bow.Find("script").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		parts := strings.Split(text, "this.resource_data        = ")
		if len(parts) != 2 {
			return
		}
		body = strings.Split(parts[1], ";\n")[0]
	})
	data := []Resource{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// APIEndpoint is the endpoint for all APIs.
const APIEndpoint = "https://piazza.com/logic/api?method="

// ContentType is the content type for all API requests.
const ContentType = "application/json; charset=UTF-8"

// APIReq is the generic wrapper for an API request.
type APIReq struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

func (c *Client) MakeAPIReq(method string, params interface{}, resp interface{}) error {
	req := APIReq{
		Method: method,
		Params: params,
	}
	url := APIEndpoint + method
	if len(c.aid) > 0 {
		url += "&aid=" + c.aid
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return err
	}
	httpReq, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	httpReq.Header.Add("Content-Type", ContentType)
	for _, c := range c.Cookies() {
		httpReq.AddCookie(c)
	}
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		return errors.Errorf("StatusCode = %d", httpResp.StatusCode)
	}

	if resp == nil {
		return nil
	}

	body, _ := ioutil.ReadAll(httpResp.Body)

	if err := json.Unmarshal(body, resp); err != nil {
		return errors.Wrapf(err, "method %q", method)
	}

	return nil
}

// EmailPrefs is the UserStatus.Result.Config subfield relating to email prefs.
// There is one extra "careers" field.
type EmailPrefs map[string]struct {
	AutoFollow interface{} `json:"auto_follow"`
	New        string      `json:"new"`
	Updates    string      `json:"updates"`
	NoEvents   bool        `json:"no_events"`
	Throttle   int         `json:"throttle"`
}

// UserStatus contains all the fields that the UserStatusAPI endpoint returns.
type UserStatus struct {
	Aid    string      `json:"aid"`
	Error  interface{} `json:"error"`
	Result struct {
		Activated            int      `json:"activated"`
		CanAdmin             struct{} `json:"can_admin"`
		CanAnonymize         bool     `json:"can_anonymize"`
		CareersAtAGlanceData struct {
			NofNumCompanies      int `json:"nof_num_companies"`
			NofNumSearches       int `json:"nof_num_searches"`
			NofNumStudentsViewed int `json:"nof_num_students_viewed"`
		} `json:"careers_at_a_glance_data"`
		Config struct {
			CareersNotifications []struct {
				Content               string `json:"content"`
				CreatedAt             int    `json:"created_at"`
				ID                    string `json:"id"`
				ImageSource           string `json:"image_source"`
				Link                  string `json:"link"`
				NotificationEventTime int    `json:"notification_event_time"`
				Status                string `json:"status"`
				Type                  string `json:"type"`
				UpdatedAt             int    `json:"updated_at"`
			} `json:"careers_notifications"`
			EmailPrefs        EmailPrefs     `json:"email_prefs"`
			EmailThrottle     map[string]int `json:"email_throttle"`
			EmailThrottleLast map[string]int `json:"email_throttle_last"`
			EnrollTime        map[string]int `json:"enroll_time"`
			F15EDigest0125    int            `json:"f15_e_digest0125"`
			Feed              struct {
				Following  int `json:"following"`
				Unread     int `json:"unread"`
				Unresolved int `json:"unresolved"`
				Updated    int `json:"updated"`
			} `json:"feed"`
			FeedDetails string `json:"feed_details"`
			FeedDigest  struct {
				State string `json:"state"`
				When  int    `json:"when"`
			} `json:"feed_digest"`
			FeedGroups                     []string `json:"feed_groups"`
			InRoster                       []string `json:"in_roster"`
			JobDropdownsLastSeen           string   `json:"job_dropdowns_last_seen"`
			LastNetworks                   []string `json:"last_networks"`
			Logins                         int      `json:"logins"`
			NoFeed                         bool     `json:"no_feed"`
			NotificationTypeLastCalculated struct {
				AppearedInSearch      int `json:"appeared_in_search"`
				ClassmatesCompanyView int `json:"classmates_company_view"`
				CompaniesOnline       int `json:"companies_online"`
				NewCompany            int `json:"new_company"`
				UpcomingEvents        int `json:"upcoming_events"`
			} `json:"notification_type_last_calculated"`
			Published           bool              `json:"published"`
			PublishedTime       string            `json:"published_time"`
			Roles               map[string]string `json:"roles"`
			SeenMessage         []string          `json:"seen_message"`
			TechtourViewCounter int               `json:"techtour_view_counter"`
			UseLite             bool              `json:"use_lite"`
			WaitlistModalType   string            `json:"waitlist_modal_type"`
			WaitlistResponse    string            `json:"waitlist_response"`
		} `json:"config"`
		Email        string   `json:"email"`
		Emails       []string `json:"emails"`
		Facebook     struct{} `json:"facebook"`
		FeedPrefetch struct {
			Avg    int         `json:"avg"`
			AvgCnt interface{} `json:"avg_cnt"`
			Draft  struct{}    `json:"draft"`
			Drafts struct{}    `json:"drafts"`
			Feed   []struct {
				BucketName    string        `json:"bucket_name"`
				BucketOrder   int           `json:"bucket_order"`
				ContentSnipet string        `json:"content_snipet"`
				Fol           string        `json:"fol"`
				Folders       []interface{} `json:"folders"`
				ID            string        `json:"id"`
				IsNew         bool          `json:"is_new"`
				Log           []struct {
					N string `json:"n"`
					T string `json:"t"`
					U string `json:"u"`
				} `json:"log"`
				M                 int      `json:"m"`
				MainVersion       int      `json:"main_version"`
				Modified          string   `json:"modified"`
				NoAnswerFollowup  int      `json:"no_answer_followup"`
				Nr                int      `json:"nr"`
				NumFavorites      int      `json:"num_favorites"`
				Pin               int      `json:"pin"`
				RequestInstructor int      `json:"request_instructor"`
				Rq                int      `json:"rq"`
				Score             float64  `json:"score"`
				Status            string   `json:"status"`
				Subject           string   `json:"subject"`
				Tags              []string `json:"tags"`
				Type              string   `json:"type"`
				UniqueViews       int      `json:"unique_views"`
				Updated           string   `json:"updated"`
				ViewAdjust        int      `json:"view_adjust"`
			} `json:"feed"`
			Hof struct {
				BestAnswer []struct {
					Nr   int         `json:"nr"`
					Text string      `json:"text"`
					Time int         `json:"time"`
					UID  interface{} `json:"uid"`
					When int         `json:"when"`
				} `json:"best_answer"`
			} `json:"hof"`
			LastNetworks         []string      `json:"last_networks"`
			More                 bool          `json:"more"`
			NoOpenTeammateSearch int           `json:"no_open_teammate_search"`
			NotificationSubjects struct{}      `json:"notification_subjects"`
			Notifications        []interface{} `json:"notifications"`
			Sort                 string        `json:"sort"`
			T                    int           `json:"t"`
			Tags                 struct {
				Instructor      []string `json:"instructor"`
				InstructorCount struct {
					Assignment1 int `json:"assignment1"`
					Lecture     int `json:"lecture"`
					Logistics   int `json:"logistics"`
					Other       int `json:"other"`
				} `json:"instructor_count"`
				InstructorUpd struct {
					Assignment1 int `json:"assignment1"`
					Lecture     int `json:"lecture"`
					Logistics   int `json:"logistics"`
					Other       int `json:"other"`
				} `json:"instructor_upd"`
				Popular      []string `json:"popular"`
				PopularCount struct {
					Assignment1 int `json:"assignment1"`
					Lecture     int `json:"lecture"`
					Logistics   int `json:"logistics"`
					Other       int `json:"other"`
				} `json:"popular_count"`
			} `json:"tags"`
			TokenData struct {
				ChannelIds []string `json:"channel_ids"`
				Signature  string   `json:"signature"`
				Timestamp  int      `json:"timestamp"`
			} `json:"token_data"`
			Users  int `json:"users"`
			Users7 int `json:"users_7"`
		} `json:"feed_prefetch"`
		ID          string   `json:"id"`
		LastContent struct{} `json:"last_content"`
		LastNetwork string   `json:"last_network"`
		Name        string   `json:"name"`
		Networks    []struct {
			Anonymity string `json:"anonymity"`
			AutoJoin  string `json:"auto_join"`
			Config    struct {
				ClassSections struct {
					AllowEnroll int      `json:"allow_enroll"`
					Sections    []string `json:"sections"`
				} `json:"class_sections"`
				DefaultPostsToPrivate bool `json:"default_posts_to_private"`
				DisableFolders        bool `json:"disable_folders"`
				DisableStudentPolls   bool `json:"disable_student_polls"`
				DisableSyntax         bool `json:"disable_syntax"`
				GetFamiliarNr         int  `json:"get_familiar_nr"`
				HasWorkAt             int  `json:"has_work_at"`
				IntroducePiazzaNr     int  `json:"introduce_piazza_nr"`
				Onboard               struct {
					AddInst int `json:"add_inst"`
					AddStud int `json:"add_stud"`
				} `json:"onboard"`
				PublicVisibilitySettings struct {
					Announcements    bool `json:"announcements"`
					ResourceSections struct {
						General           bool `json:"general"`
						Homework          bool `json:"homework"`
						HomeworkSolutions bool `json:"homework_solutions"`
						LectureNotes      bool `json:"lecture_notes"`
					} `json:"resource_sections"`
				} `json:"public_visibility_settings"`
				RegUserCount     int `json:"reg_user_count"`
				ResourceSections []struct {
					DateTitle  string `json:"date_title"`
					HasDate    bool   `json:"has_date"`
					Name       string `json:"name"`
					Title      string `json:"title"`
					Visibility bool   `json:"visibility"`
				} `json:"resource_sections"`
				Roles struct {
					Admin struct {
						AdminRoster             bool `json:"admin_roster"`
						CanPostAnonymousAll     bool `json:"can_post_anonymous_all"`
						CanPostAnonymousMembers bool `json:"can_post_anonymous_members"`
						ExpertAnswerCreate      bool `json:"expert_answer_create"`
						ExpertAnswerEdit        bool `json:"expert_answer_edit"`
						ExpertAnswerEndorse     bool `json:"expert_answer_endorse"`
						FollowupEdit            bool `json:"followup_edit"`
						ManageFolders           bool `json:"manage_folders"`
						ManageGroupInfo         bool `json:"manage_group_info"`
						ManageGroups            bool `json:"manage_groups"`
						ManageResources         bool `json:"manage_resources"`
						MemberAnswerCreate      bool `json:"member_answer_create"`
						MemberAnswerEdit        bool `json:"member_answer_edit"`
						MemberAnswerEndorse     bool `json:"member_answer_endorse"`
						MemberRoster            bool `json:"member_roster"`
						NewFollowup             bool `json:"new_followup"`
						NewPost                 bool `json:"new_post"`
						QuestionDelete          bool `json:"question_delete"`
						QuestionEdit            bool `json:"question_edit"`
					} `json:"admin"`
					Instructor struct {
						AdminRoster         bool `json:"admin_roster"`
						ExpertAnswerCreate  bool `json:"expert_answer_create"`
						ExpertAnswerEdit    bool `json:"expert_answer_edit"`
						ExpertAnswerEndorse bool `json:"expert_answer_endorse"`
						FollowupEdit        bool `json:"followup_edit"`
						ManageFolders       bool `json:"manage_folders"`
						ManageGroupInfo     bool `json:"manage_group_info"`
						ManageGroups        bool `json:"manage_groups"`
						ManageResources     bool `json:"manage_resources"`
						MemberAnswerEdit    bool `json:"member_answer_edit"`
						MemberAnswerEndorse bool `json:"member_answer_endorse"`
						MemberRoster        bool `json:"member_roster"`
						NewFollowup         bool `json:"new_followup"`
						NewPost             bool `json:"new_post"`
						QuestionDelete      bool `json:"question_delete"`
						QuestionEdit        bool `json:"question_edit"`
					} `json:"instructor"`
					Professor struct {
						AdminRoster         bool `json:"admin_roster"`
						ExpertAnswerCreate  bool `json:"expert_answer_create"`
						ExpertAnswerEdit    bool `json:"expert_answer_edit"`
						ExpertAnswerEndorse bool `json:"expert_answer_endorse"`
						FollowupEdit        bool `json:"followup_edit"`
						ManageFolders       bool `json:"manage_folders"`
						ManageGroupInfo     bool `json:"manage_group_info"`
						ManageGroups        bool `json:"manage_groups"`
						ManageResources     bool `json:"manage_resources"`
						MemberAnswerEdit    bool `json:"member_answer_edit"`
						MemberAnswerEndorse bool `json:"member_answer_endorse"`
						MemberRoster        bool `json:"member_roster"`
						NewFollowup         bool `json:"new_followup"`
						NewPost             bool `json:"new_post"`
						QuestionDelete      bool `json:"question_delete"`
						QuestionEdit        bool `json:"question_edit"`
					} `json:"professor"`
					Student struct {
						CanPostAnonymousMembers bool `json:"can_post_anonymous_members"`
						ExpertAnswerEndorse     bool `json:"expert_answer_endorse"`
						MemberAnswerCreate      bool `json:"member_answer_create"`
						MemberAnswerEdit        bool `json:"member_answer_edit"`
						MemberAnswerEndorse     bool `json:"member_answer_endorse"`
						NewFollowup             bool `json:"new_followup"`
						NewPost                 bool `json:"new_post"`
						QuestionEdit            bool `json:"question_edit"`
					} `json:"student"`
					Ta struct {
						AdminRoster         bool `json:"admin_roster"`
						ExpertAnswerCreate  bool `json:"expert_answer_create"`
						ExpertAnswerEdit    bool `json:"expert_answer_edit"`
						ExpertAnswerEndorse bool `json:"expert_answer_endorse"`
						FollowupEdit        bool `json:"followup_edit"`
						ManageFolders       bool `json:"manage_folders"`
						ManageGroupInfo     bool `json:"manage_group_info"`
						ManageGroups        bool `json:"manage_groups"`
						ManageResources     bool `json:"manage_resources"`
						MemberAnswerEdit    bool `json:"member_answer_edit"`
						MemberAnswerEndorse bool `json:"member_answer_endorse"`
						MemberRoster        bool `json:"member_roster"`
						NewFollowup         bool `json:"new_followup"`
						NewPost             bool `json:"new_post"`
						QuestionDelete      bool `json:"question_delete"`
						QuestionEdit        bool `json:"question_edit"`
					} `json:"ta"`
				} `json:"roles"`
				SeenMessage  []string `json:"seen_message"`
				TipsTricksNr int      `json:"tips_tricks_nr"`
			} `json:"config"`
			CourseDescription  string         `json:"course_description"`
			CourseNumber       string         `json:"course_number"`
			CreatedAt          string         `json:"created_at"`
			CreatorName        string         `json:"creator_name"`
			Department         string         `json:"department"`
			EndDate            interface{}    `json:"end_date"`
			Enrollment         interface{}    `json:"enrollment"`
			Folders            []string       `json:"folders"`
			GeneralInformation []interface{}  `json:"general_information"`
			ID                 string         `json:"id"`
			IsOpen             bool           `json:"isOpen"`
			MyName             string         `json:"my_name"`
			Name               string         `json:"name"`
			OfficeHours        struct{}       `json:"office_hours"`
			PrivatePosts       string         `json:"private_posts"`
			ProfHash           map[string]int `json:"prof_hash"`
			Profs              []struct {
				Admin           bool        `json:"admin"`
				AdminPermission int         `json:"admin_permission"`
				ClassSections   []string    `json:"class_sections"`
				Email           string      `json:"email"`
				FacebookID      interface{} `json:"facebook_id"`
				ID              string      `json:"id"`
				Name            string      `json:"name"`
				Photo           interface{} `json:"photo"`
				Role            string      `json:"role"`
				Us              bool        `json:"us"`
			} `json:"profs"`
			School           string        `json:"school"`
			SchoolEmails     string        `json:"school_emails"`
			SchoolExt        string        `json:"school_ext"`
			SchoolID         string        `json:"school_id"`
			SchoolShort      string        `json:"school_short"`
			ShortNumber      string        `json:"short_number"`
			SpecialTags      []interface{} `json:"special_tags"`
			StartDate        string        `json:"start_date"`
			Status           string        `json:"status"`
			Syllabus         string        `json:"syllabus"`
			Taxonomy         []interface{} `json:"taxonomy"`
			Term             string        `json:"term"`
			Topics           []string      `json:"topics"`
			TotalContentProf int           `json:"total_content_prof"`
			TotalContentStud int           `json:"total_content_stud"`
			Type             string        `json:"type"`
			UserCount        int           `json:"user_count"`
		} `json:"networks"`
		NewQuestions  map[string]int `json:"new_questions"`
		Photo         string         `json:"photo"`
		PhotoOriginal string         `json:"photo_original"`
		Profile       struct{}       `json:"profile"`
		Sid           string         `json:"sid"`
	} `json:"result"`
}

// UserStatus returns the user status.
func (c *Client) UserStatus() (UserStatus, error) {
	var resp UserStatus
	if err := c.MakeAPIReq("user.status", struct{}{}, &resp); err != nil {
		return UserStatus{}, err
	}
	c.aid = resp.Aid
	return resp, nil
}

type updateEmailsReq struct {
	EmailPrefs EmailPrefs `json:"email_prefs"`
}

// OptOutOfEmails sets `new: "no-emails"` on all courses.
func (c *Client) OptOutOfEmails() error {
	status, err := c.UserStatus()
	if err != nil {
		return err
	}
	prefs := status.Result.Config.EmailPrefs
	delete(prefs, "career")
	for c, pref := range prefs {
		pref.New = "no-emails"
		prefs[c] = pref
	}
	req := updateEmailsReq{prefs}
	if err := c.MakeAPIReq("user.update", req, nil); err != nil {
		return err
	}
	return nil
}

// Cookies returns the cookies for the Piazza client.
func (c *Client) Cookies() []*http.Cookie {
	return c.bow.SiteCookies()
}
