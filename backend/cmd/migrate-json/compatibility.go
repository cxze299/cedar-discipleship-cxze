package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var retroDatePattern = regexp.MustCompile(`(?:补签】?|补卡】?)\s*(\d{1,2})月(\d{1,2})日`)

var bibleBookAliases = map[string]string{
	"创": "创世记", "出": "出埃及记", "利": "利未记", "民": "民数记", "申": "申命记",
	"书": "约书亚记", "士": "士师记", "得": "路得记", "撒上": "撒母耳记上", "撒下": "撒母耳记下",
	"王上": "列王纪上", "王下": "列王纪下", "代上": "历代志上", "代下": "历代志下",
	"拉": "以斯拉记", "尼": "尼希米记", "斯": "以斯帖记", "伯": "约伯记", "诗": "诗篇",
	"箴": "箴言", "传": "传道书", "歌": "雅歌", "赛": "以赛亚书", "耶": "耶利米书",
	"哀": "耶利米哀歌", "结": "以西结书", "但": "但以理书", "何": "何西阿书", "珥": "约珥书",
	"摩": "阿摩司书", "俄": "俄巴底亚书", "拿": "约拿书", "弥": "弥迦书", "鸿": "那鸿书",
	"哈": "哈巴谷书", "番": "西番雅书", "该": "哈该书", "亚": "撒迦利亚书", "玛": "玛拉基书",
	"太": "马太福音", "可": "马可福音", "路": "路加福音", "约": "约翰福音", "徒": "使徒行传",
	"罗": "罗马书", "林前": "哥林多前书", "林后": "哥林多后书", "加": "加拉太书",
	"弗": "以弗所书", "腓": "腓立比书", "西": "歌罗西书", "帖前": "帖撒罗尼迦前书",
	"帖后": "帖撒罗尼迦后书", "提前": "提摩太前书", "提后": "提摩太后书", "多": "提多书",
	"门": "腓利门书", "来": "希伯来书", "雅": "雅各书", "彼前": "彼得前书",
	"彼后": "彼得后书", "约壹": "约翰一书", "约贰": "约翰二书", "约叁": "约翰三书",
	"犹": "犹大书", "启": "启示录",
}

func (r *oldRecord) UnmarshalJSON(data []byte) error {
	type recordAlias oldRecord
	var english recordAlias
	if err := json.Unmarshal(data, &english); err != nil {
		return err
	}
	*r = oldRecord(english)
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.ID = firstInt64(r.ID, raw["Id"], raw["ID"])
	r.Name = firstNonEmpty(r.Name, stringValue(raw["姓名"]))
	r.CheckinTime = firstNonEmpty(r.CheckinTime, stringValue(raw["打卡时间"]))
	r.LogicalDate = firstNonEmpty(r.LogicalDate, stringValue(raw["逻辑日期"]))
	r.IsRetro = firstAny(r.IsRetro, raw["是否补签"])
	r.Daily = firstNonEmpty(r.Daily, stringValue(raw["每日灵修"]))
	r.Scripture = firstNonEmpty(r.Scripture, stringValue(raw["每日读经"]))
	r.Weekly = firstNonEmpty(r.Weekly, stringValue(raw["周任务"]))
	r.Book = firstNonEmpty(r.Book, stringValue(raw["周读物"]))
	r.Video = firstNonEmpty(r.Video, stringValue(raw["周视频"]))
	r.Verse = firstNonEmpty(r.Verse, stringValue(raw["周背经"]))
	r.Detail = firstNonEmpty(r.Detail, stringValue(raw["打卡详情"]))
	r.Note = firstNonEmpty(r.Note, stringValue(raw["备注"]), stringValue(raw["分享记录"]))
	r.Kind = firstNonEmpty(r.Kind, stringValue(raw["类型"]))
	r.Part = firstNonEmpty(r.Part, stringValue(raw["部分"]))
	if r.LogicalDate == "" {
		r.LogicalDate = logicalDateFromLegacyRecord(*r)
	}
	return nil
}

func normalizeLegacyConfig(cfg *oldConfig, opt options) error {
	if cfg == nil {
		return nil
	}
	for name, value := range map[string]string{
		"daily checkin mode":  opt.dailyCheckinMode,
		"weekly checkin mode": opt.weeklyCheckinMode,
		"devotion mode":       opt.devotionMode,
	} {
		if value == "" {
			continue
		}
		valid := (name == "daily checkin mode" && (value == "combined" || value == "separate")) ||
			(name == "weekly checkin mode" && (value == "per_reading" || value == "aggregate")) ||
			(name == "devotion mode" && (value == "auto" || value == "numbered" || value == "date"))
		if !valid {
			return fmt.Errorf("invalid %s %q", name, value)
		}
	}

	var sections map[string]any
	if len(cfg.TaskSections) > 0 {
		if err := json.Unmarshal(cfg.TaskSections, &sections); err != nil {
			return err
		}
	}
	if sections == nil {
		sections = map[string]any{}
	}
	daily := ensureMap(sections, "daily")
	devotion := ensureMap(daily, "devotion")
	if devotion["path"] == nil && daily["path"] != nil {
		devotion["path"] = daily["path"]
	}
	if opt.devotionMode != "" {
		devotion["mode"] = opt.devotionMode
	}
	if devotion["mode"] == nil {
		devotion["mode"] = "auto"
	}

	scripture := ensureMap(daily, "scripture")
	if len(cfg.DailyReading) > 0 {
		if err := normalizeDailyReading(cfg.DailyReading, scripture); err != nil {
			return err
		}
	}
	if opt.dailyCheckinMode != "" {
		daily["checkin_mode"] = opt.dailyCheckinMode
	} else if daily["checkin_mode"] == nil {
		daily["checkin_mode"] = "combined"
	}
	if daily["checkin_mode"] == "separate" && len(cfg.DailyReading) == 0 {
		scripture["enabled"] = true
		scripture["type"] = "checkin"
		delete(scripture, "book")
		delete(scripture, "book_id")
		delete(scripture, "books")
		delete(scripture, "sequence")
	}

	weekly := ensureMap(sections, "weekly")
	mode := opt.weeklyCheckinMode
	if mode == "" && numberValue(rawMap(cfg.AdminUI)["max_reading_slots"]) == 1 {
		mode = "aggregate"
	}
	if mode == "" {
		mode = "per_reading"
	}
	weekly["checkin_mode"] = mode
	if mode == "aggregate" {
		weekly["checkin_enabled"] = true
	}
	readingPath := stringValue(weekly["reading_path"])
	for i := range cfg.WeeklySchedule {
		cfg.WeeklySchedule[i].WeeklyCheckin = mode == "aggregate"
		cfg.WeeklySchedule[i].ReadingPath = readingPath
		for j := range cfg.WeeklySchedule[i].Readings {
			ref := &cfg.WeeklySchedule[i].Readings[j]
			if ref.Type == "" && isExternalContentURL(ref.URL) &&
				(strings.HasSuffix(strings.ToLower(ref.URL), ".htm") || strings.Contains(strings.ToLower(ref.URL), ".htm?")) {
				ref.Type = "iframe"
			}
		}
	}
	normalized, err := json.Marshal(sections)
	if err != nil {
		return err
	}
	cfg.TaskSections = normalized
	return nil
}

func normalizeDailyReading(raw json.RawMessage, scripture map[string]any) error {
	var source struct {
		Base struct {
			Date    string `json:"date"`
			Book    string `json:"book"`
			Chapter int    `json:"chapter"`
		} `json:"base"`
		Books          [][]any `json:"books"`
		ChaptersPerDay int     `json:"chapters_per_day"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return err
	}
	startBook, ok := canonicalBibleBook(source.Base.Book)
	if !ok {
		return fmt.Errorf("unknown daily reading book %q", source.Base.Book)
	}
	scripture["enabled"] = true
	scripture["type"] = "sequence"
	scripture["start_date"] = source.Base.Date
	scripture["book"] = startBook.Book
	scripture["book_id"] = startBook.BookID
	scripture["start_chapter"] = source.Base.Chapter
	if source.ChaptersPerDay < 1 {
		source.ChaptersPerDay = numberValue(scripture["chapters_per_day"])
	}
	if source.ChaptersPerDay < 1 {
		source.ChaptersPerDay = 1
	}
	scripture["chapters_per_day"] = source.ChaptersPerDay
	books := make([]any, 0, len(source.Books))
	for _, item := range source.Books {
		if len(item) < 2 {
			return errors.New("invalid daily reading book entry")
		}
		name := stringValue(item[0])
		book, ok := canonicalBibleBook(name)
		if !ok {
			return fmt.Errorf("unknown daily reading book %q", name)
		}
		chapters := numberValue(item[1])
		if chapters < 1 {
			return fmt.Errorf("invalid chapter count for %q", name)
		}
		books = append(books, map[string]any{
			"book": book.Book, "book_id": book.BookID, "chapters": chapters,
		})
	}
	scripture["books"] = books
	scripture["sequence"] = books
	return nil
}

func learningSettings(cfg oldConfig) (map[string]json.RawMessage, error) {
	settings := map[string]json.RawMessage{}
	if len(cfg.TaskSections) > 0 {
		sections, err := normalizeTaskSections(cfg.TaskSections)
		if err != nil {
			return nil, err
		}
		settings["task_sections"] = sections
	}
	for key, value := range map[string]json.RawMessage{
		"mounted_files":          cfg.MountedFiles,
		"weekly_reading_catalog": cfg.WeeklyReadingCatalog,
		"class_rep_shares":       cfg.ClassRepShares,
		"admin_ui":               cfg.AdminUI,
	} {
		if len(value) > 0 && string(value) != "null" {
			settings[key] = value
		}
	}
	return settings, nil
}

func canonicalBibleBook(value string) (scriptureBook, bool) {
	value = strings.TrimSpace(value)
	if full := bibleBookAliases[value]; full != "" {
		value = full
	}
	for _, book := range bibleBooks {
		if book.Book == value {
			return book, true
		}
	}
	return scriptureBook{}, false
}

func logicalDateFromLegacyRecord(record oldRecord) string {
	checked, err := parseTime(record.CheckinTime)
	if err != nil {
		return ""
	}
	if match := retroDatePattern.FindStringSubmatch(record.Detail); len(match) == 3 {
		month, day := atoiOrZero(match[1]), atoiOrZero(match[2])
		candidate := time.Date(checked.Year(), time.Month(month), day, 0, 0, 0, 0, checked.Location())
		if candidate.After(checked.AddDate(0, 1, 0)) {
			candidate = candidate.AddDate(-1, 0, 0)
		}
		return candidate.Format("2006-01-02")
	}
	return checked.Format("2006-01-02")
}

func ensureMap(parent map[string]any, key string) map[string]any {
	child := mapValue(parent, key)
	if child == nil {
		child = map[string]any{}
		parent[key] = child
	}
	return child
}

func rawMap(raw json.RawMessage) map[string]any {
	var value map[string]any
	_ = json.Unmarshal(raw, &value)
	return value
}

func firstAny(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstInt64(fallback int64, values ...any) int64 {
	if fallback != 0 {
		return fallback
	}
	for _, value := range values {
		switch value := value.(type) {
		case float64:
			return int64(value)
		case json.Number:
			number, _ := value.Int64()
			return number
		}
	}
	return 0
}
