package common

import (
	"database/sql"
	"hack-browser-data/log"
	"hack-browser-data/utils"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tidwall/gjson"
)

const (
	Chrome = "Chrome"
	Safari = "Safari"
)

var (
	FullData      = new(BrowserData)
	bookmarkList  []*Bookmarks
	cookieList    []*Cookies
	historyList   []*History
	loginItemList []*LoginData
)

const (
	bookmarkID       = "id"
	bookmarkAdded    = "date_added"
	bookmarkUrl      = "url"
	bookmarkName     = "name"
	bookmarkType     = "type"
	bookmarkChildren = "children"
)

type (
	BrowserData struct {
		BrowserName string
		LoginData   []*LoginData
		Bookmarks   []*Bookmarks
		Cookies     []*Cookies
		History     []*History
	}
	LoginData struct {
		UserName    string    `json:"user_name"`
		encryptPass []byte    `json:"-"`
		Password    string    `json:"password"`
		LoginUrl    string    `json:"login_url"`
		CreateDate  time.Time `json:"create_date"`
	}
	Bookmarks struct {
		ID        string    `json:"id"`
		DateAdded time.Time `json:"date_added"`
		URL       string    `json:"url"`
		Name      string    `json:"name"`
		Type      string    `json:"type"`
	}
	Cookies struct {
		KeyName      string
		encryptValue []byte
		Value        string
		Host         string
		Path         string
		IsSecure     bool
		IsHTTPOnly   bool
		HasExpire    bool
		IsPersistent bool
		CreateDate   time.Time
		ExpireDate   time.Time
	}
	History struct {
		Url           string
		Title         string
		VisitCount    int
		LastVisitTime time.Time
	}
)

func ParseDB(dbname string) {
	switch dbname {
	case utils.LoginData:
		parseLogin()
	case utils.Bookmarks:
		parseBookmarks()
	case utils.Cookies:
		parseCookie()
	case utils.History:
		parseHistory()
	}
}

func parseBookmarks() {
	bookmarks, err := utils.ReadFile(utils.Bookmarks)
	if err != nil {
		log.Println(err)
	}
	r := gjson.Parse(bookmarks)
	if r.Exists() {
		roots := r.Get("roots")
		roots.ForEach(func(key, value gjson.Result) bool {
			getBookmarkChildren(value)
			return true
		})
	}
	FullData.Bookmarks = bookmarkList
}

var queryLogin = `SELECT origin_url, username_value, password_value, date_created FROM logins`

func parseLogin() {
	//datetime(visit_time / 1000000 + (strftime('%s', '1601-01-01')), 'unixepoch')
	login := &LoginData{}
	loginDB, err := sql.Open("sqlite3", utils.LoginData)
	defer func() {
		if err := loginDB.Close(); err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		log.Println(err)
	}
	err = loginDB.Ping()
	rows, err := loginDB.Query(queryLogin)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()
	for rows.Next() {
		var (
			url, username, password string
			pwd                     []byte
			create                  int64
		)
		err = rows.Scan(&url, &username, &pwd, &create)
		login = &LoginData{
			UserName:    username,
			encryptPass: pwd,
			LoginUrl:    url,
			CreateDate:  utils.TimeEpochFormat(create),
		}
		if len(pwd) > 3 {
			// remove prefix 'v10'
			password, err = utils.Aes128CBCDecrypt(pwd[3:])
		}
		login.Password = password
		if err != nil {
			log.Println(err)
		}
		loginItemList = append(loginItemList, login)
	}
	FullData.LoginData = loginItemList
}

var queryCookie = `SELECT name, encrypted_value, host_key, path, creation_utc, expires_utc, is_secure, is_httponly, has_expires, is_persistent FROM cookies`

func parseCookie() {
	cookies := &Cookies{}
	cookieDB, err := sql.Open("sqlite3", utils.Cookies)
	defer func() {
		if err := cookieDB.Close(); err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		log.Println(err)
	}
	err = cookieDB.Ping()
	rows, err := cookieDB.Query(queryCookie)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()
	for rows.Next() {
		var (
			key, host, path, value                        string
			isSecure, isHTTPOnly, hasExpire, isPersistent int
			createDate, expireDate                        int64
			encryptValue                                  []byte
		)
		err = rows.Scan(&key, &encryptValue, &host, &path, &createDate, &expireDate, &isSecure, &isHTTPOnly, &hasExpire, &isPersistent)
		cookies = &Cookies{
			KeyName:      key,
			Host:         host,
			Path:         path,
			encryptValue: encryptValue,
			IsSecure:     utils.IntToBool(isSecure),
			IsHTTPOnly:   utils.IntToBool(isHTTPOnly),
			HasExpire:    utils.IntToBool(hasExpire),
			IsPersistent: utils.IntToBool(isPersistent),
			CreateDate:   utils.TimeEpochFormat(createDate),
			ExpireDate:   utils.TimeEpochFormat(expireDate),
		}
		if len(encryptValue) > 3 {
			// remove prefix 'v10'
			value, err = utils.Aes128CBCDecrypt(encryptValue[3:])
		}
		cookies.Value = value
		cookieList = append(cookieList, cookies)
	}
	FullData.Cookies = cookieList
}

var queryUrl = `SELECT url, title, visit_count, last_visit_time FROM urls`

func parseHistory() {
	history := &History{}
	historyDB, err := sql.Open("sqlite3", utils.History)
	defer func() {
		if err := historyDB.Close(); err != nil {
			log.Println(err)
		}
	}()
	if err != nil {
		log.Println(err)
	}
	err = historyDB.Ping()
	rows, err := historyDB.Query(queryUrl)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
		}
	}()
	for rows.Next() {
		var (
			url, title    string
			visitCount    int
			lastVisitTime int64
		)
		err := rows.Scan(&url, &title, &visitCount, &lastVisitTime)
		history = &History{
			Url:           url,
			Title:         title,
			VisitCount:    visitCount,
			LastVisitTime: utils.TimeEpochFormat(lastVisitTime),
		}
		if err != nil {
			log.Println(err)
			continue
		}
		historyList = append(historyList, history)
	}
	FullData.History = historyList
}

func getBookmarkChildren(value gjson.Result) (children gjson.Result) {
	b := new(Bookmarks)
	b.ID = value.Get(bookmarkID).String()
	nodeType := value.Get(bookmarkType)
	b.DateAdded = utils.TimeEpochFormat(value.Get(bookmarkAdded).Int())
	b.URL = value.Get(bookmarkUrl).String()
	b.Name = value.Get(bookmarkName).String()
	children = value.Get(bookmarkChildren)
	if nodeType.Exists() {
		b.Type = nodeType.String()
		bookmarkList = append(bookmarkList, b)
		if children.Exists() && children.IsArray() {
			for _, v := range children.Array() {
				children = getBookmarkChildren(v)
			}
		}
	}
	return children
}
