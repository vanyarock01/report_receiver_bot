package report_receiver_bot

import (
    "fmt"
    "log"
    "time"
    "errors"
    "strings"
    "io/ioutil"
    "path/filepath"
    "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
    utfOkEmoji  = "\xE2\x9C\x85"
    utfNotEmoji = "\xE2\x9D\x8C"
)

type ReportReceiver struct {
    Config     *Config
    Bot        *tgbotapi.BotAPI
    configPath string
}

func (rc *ReportReceiver) NotifyGroupAdmin(stud *Student, work string, fileID string) {
    log.Printf("[info] Notify report from %s to Group admin", stud.UserName)
    admin := rc.Config.GetAdminByGroup(stud.GroupName)
    if admin == nil {
        log.Printf("[info] Group %s admin not found", )
        return
    }
    if admin.ChatID == 0 {
        log.Printf("[info] Admin %s not linked", admin.User)
        return
    }
    req := tgbotapi.NewDocumentShare(admin.ChatID, fileID)
    req.Caption = fmt.Sprintf("[%s] %s %s", work, stud.FirstName, stud.LastName)
    rc.Bot.Send(req)
}

func (rc *ReportReceiver) UnauthorizedHandler(update *tgbotapi.Update) {
    // send a funny picture with cat to a passerby 
    msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, "forbidden")
    msg.FileID = "https://http.cat/403"
    msg.UseExisting = true
    rc.Bot.Send(msg)
}

func (rc *ReportReceiver) HelpCommandHandler(update *tgbotapi.Update) {
    workList := ""
    for _, work := range rc.Config.Work {
        workList += fmt.Sprintf("• \"%s\"\n", work)
    }
    bt := "`"
    man := fmt.Sprintf(`*How to use this bot?*

Create message with attached file and text _<work name>_.
*ONE MESSAGE - ONE FILE*.

WORK NAMES:
%s
If you sent only one file (for example %spdf%s) you will receive a notification that you need to send %sdocx%s file.

• For show this message run command %s/help%s
• For show statistic run command %s/stat%s`, workList, bt, bt, bt, bt, bt, bt, bt, bt)

    msg := tgbotapi.NewMessage(update.Message.Chat.ID, man)
    msg.ParseMode = "markdown"
    rc.Bot.Send(msg)
}


func (rc *ReportReceiver) StatCommandHandler(stud *Student, update *tgbotapi.Update) {
    log.Printf("[info] Collect statistic for %s", stud.UserName)

    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unhandled behavior???")
    defer func() {
        rc.Bot.Send(msg)
    }()

    text := fmt.Sprintf("*Student*: _%s %s %s_\n*Group*: _%s_\n",
                        stud.LastName,
                        stud.FirstName,
                        stud.SecondName,
                        stud.GroupName)

    for _, workName := range rc.Config.Work {
        workPath := filepath.Join(rc.Config.WorkDir, stud.WorkDir, workName)
        reports, err := ioutil.ReadDir(workPath)
        if err != nil {
            log.Printf("[error] Error during ReadDir %s", workPath)
            msg.Text = "Internal error: wait some time and retry"

            return
        }

        reportMap := map[string]string{}
        for _, report := range rc.Config.Report {
            reportMap[report.Format] = utfNotEmoji
        }

        for _, report := range reports {
            ext := filepath.Ext(report.Name())
            if _, ok := reportMap[ext]; ok {
                reportMap[ext] = utfOkEmoji
            }
        }
        for _, report := range rc.Config.Report {
            text += fmt.Sprintf("| _%s:_ *%s* ", strings.ToUpper(report.Format), reportMap[report.Format]) 
        }
        text += fmt.Sprintf("| *%s*\n", workName)

    }

    msg.Text = text
    msg.ParseMode = "markdown"
}

func (rc *ReportReceiver) ReceiveDocumentHandler(stud *Student, update *tgbotapi.Update) {
    log.Printf("[info] Start document receive from %s", stud.UserName)

    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Unhandled behavior???")
    msg.ReplyToMessageID = update.Message.MessageID

    defer func() {
        rc.Bot.Send(msg)
    }()

    if update.Message.Document == nil {
        log.Printf("[info] No attached documents")
        msg.Text = "Please, pin report in docx/pdf format."

        return
    }

    docExt := filepath.Ext(update.Message.Document.FileName)
    report, err := func() (*ReportType, error) { // file validator
        for i, report := range rc.Config.Report {
            if docExt == report.Format {
                if update.Message.Document.FileSize > report.MaxSizeMb * 1024 * 1024 {
                    msg.Text = fmt.Sprintf("%s file size must be less than %dMb.", report.Format, report.MaxSizeMb)
                    return nil, errors.New("Large file")
                } else {
                    return &rc.Config.Report[i], nil
                }
            }
        }
        msg.Text = fmt.Sprintf("Please, pin report in valid format. Now [%s].", docExt)
        return nil, errors.New("No attached documents")
    }()

    if err != nil {
        log.Printf("[info] %v", err)
        return
    }

    workName, err := func() (string, error) {
        name := strings.TrimSpace(update.Message.Caption)
        for _, w := range rc.Config.Work {
            if name == w {
                return name, nil
            }
        }
        return "", errors.New("Unknown work title")
    }()

    if err != nil {
        log.Printf("[info] %s", err)
        msg.Text = "Unknown work title. Please, enter the correct work name."

        return
    }

    // download file
    fileURL, err := rc.Bot.GetFileDirectURL(update.Message.Document.FileID)
    if err != nil {
        log.Printf("[error] GetFileDirectURL: %s", err)
        msg.Text = "Internal error: wait some time and resend file"

        return
    }

    filePath := filepath.Join(stud.WorkDir, workName, fmt.Sprintf("report%s", docExt))
    err = DownloadFile(fileURL, filePath)
    if err != nil {
        log.Printf("[error] During download and save report: %s", err)
        msg.Text = "Internal error: wait some time and resend file"

        return
    }
    msg.Text = "Saved"

    // Notify group admin
    if report.Notify == true {
        go rc.NotifyGroupAdmin(stud, workName, update.Message.Document.FileID)
    }
    go rc.NotifyForgottenReportHandler(stud, workName, update.Message.Chat.ID)
}

func (rc *ReportReceiver) NotifyForgottenReportHandler(stud *Student, work string, chatID int64) {
    log.Printf("[info] Run notify scheduler for student: %s and work: %s", stud.UserName, work)
    Schedule(30 * time.Second, 5, func() (bool, error) {
        workPath := filepath.Join(rc.Config.WorkDir, stud.WorkDir, work)
        reports, err := ioutil.ReadDir(workPath)
        if err != nil {
            return false, fmt.Errorf("Fail during ioutil.ReadDir from %s: %v", workPath, err)
        }

        reportMap := map[string]string{}
        for _, report := range rc.Config.Report {
            reportMap[report.Format] = utfNotEmoji
        }

        for _, report := range reports {
            ext := filepath.Ext(report.Name())
            if _, ok := reportMap[ext]; ok {
                reportMap[ext] = utfOkEmoji
            }
        }
        // check, if all reports saved return ok
        ok := func() bool {
            for _, v := range reportMap {
                if v != utfOkEmoji {
                    return false
                }
            }
            return true
        }()
        if ok {
            // To do nothing
            return true, nil
        }
        text := fmt.Sprintf("Please, send missing report: *[%s]* ", work)
        for _, report := range rc.Config.Report {
            text += fmt.Sprintf(" _%s:_ *%s* |", strings.ToUpper(report.Format), reportMap[report.Format]) 
        }
        text += "\n"

        req := tgbotapi.NewMessage(chatID, text)
        req.ParseMode = "markdown"
        rc.Bot.Send(req)

        return false, nil
    })
}

func (rc *ReportReceiver) AdminHandler(admin *Admin, update *tgbotapi.Update) {
    if admin.ChatID != update.Message.Chat.ID {
        log.Printf("[info] Change admin [%s] chat ID [%d => %d]", admin.User, admin.ChatID, update.Message.Chat.ID)
        admin.ChatID = update.Message.Chat.ID
        rc.Config.Save(rc.configPath)
    }
}

func NewReportReceiver(config *Config, path string) (*ReportReceiver, error) {
    bot, err := tgbotapi.NewBotAPI(config.BotToken)
    if err != nil {
        return nil, fmt.Errorf("Error during bot init: %v", err)
    }

    // bot.Debug = true

    return &ReportReceiver{
        Bot: bot,
        Config: config,
        configPath: path,
    }, nil
}

func Loop(confPath string) {
    config, err := ReadConfig(confPath)
    if err != nil {
        log.Panic("[panic] Error during prepare config: %v", err)
    }
    log.Print("[info] Prepare student workdirs")
    config.PrepareStudentWorkDir()

    rc, err := NewReportReceiver(config, confPath)
    if err != nil {
        log.Panic("[panic] %v", err)
    }

    log.Printf("[info] Authorized on account %s", rc.Bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, err := rc.Bot.GetUpdatesChan(u)
    if err != nil {
        log.Panic("[panic] Error during get updates chan: %v", err)
    }

    for update := range updates {
        if update.Message == nil {
            continue
        }

        userName := update.Message.From.UserName
        log.Printf("[info] Receive message from [%s]", userName)

        admin := rc.Config.GetAdmin(userName)
        if admin != nil {
            rc.AdminHandler(admin, &update)
            // continue
        }

        stud := rc.Config.GetStudent(userName)
        if stud == nil {
            log.Printf("[warning] %s", err) 
            go rc.UnauthorizedHandler(&update)
            continue
        }

        switch cmd := update.Message.Command(); cmd {
        case "help":
            go rc.HelpCommandHandler(&update)
            continue
        case "stat":
            go rc.StatCommandHandler(stud, &update)
            continue
        default:
            // handle message
        }
        go rc.ReceiveDocumentHandler(stud, &update)
    }
}
