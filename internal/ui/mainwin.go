package ui

import (
	"context"
	"github.com/GontikR99/buffbot/internal/storage"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"os"
	"regexp"
)

var (
	uiFilePattern = regexp.MustCompile("^UI_([A-Za-z]+_[A-Za-z]+).ini$")
)

func validateEqDir(directory string) bool {
	if _, err := os.Stat(directory + "\\eqclient.ini"); err != nil {
		return false
	}
	if _, err := os.Stat(directory + "\\eqgame.exe"); err != nil {
		return false
	}
	return true
}

type lineEditHolder struct {
	Entry *walk.LineEdit
}

type mainWindowModel struct {
	mainWindow *walk.MainWindow

	dirEdit   *walk.LineEdit
	dirBrowse *walk.PushButton
	confItems map[string]*lineEditHolder

	startButton *walk.PushButton
	started     bool
}

// Figure out what portions of the GUI should be enabled, and enable them
func (mwm *mainWindowModel) shade() {
	if mwm.started {
		mwm.dirEdit.SetEnabled(false)
		mwm.dirBrowse.SetEnabled(false)
		mwm.startButton.SetEnabled(true)

		for _, v := range mwm.confItems {
			v.Entry.SetEnabled(false)
		}

		return
	} else {
		mwm.dirEdit.SetEnabled(true)
		mwm.dirBrowse.SetEnabled(true)
		for _, v := range mwm.confItems {
			v.Entry.SetEnabled(true)
		}
	}
	selectedDir := mwm.dirEdit.Text()
	validDir := validateEqDir(selectedDir)

	if validDir {
		mwm.startButton.SetEnabled(true)
	} else {
		mwm.startButton.SetEnabled(false)
	}
}

func onEqDirChanged(model *mainWindowModel, config storage.ControllerConfig) {
	dirChoice := model.dirEdit.Text()
	if dirChoice == config.EverQuestDirectory() {
		return
	}
	config.SetEverQuestDirectory(dirChoice)
	model.shade()
}

func AddConfItem(model *mainWindowModel, gb *GroupBox, ci ConfigurationLine, config storage.ControllerConfig) {
	model.confItems[ci.Name] = &lineEditHolder{}
	(*gb).Children = append((*gb).Children, Label{Text: ci.DisplayName, TextAlignment: AlignFar})
	(*gb).Children = append((*gb).Children, LineEdit{
		AssignTo: &model.confItems[ci.Name].Entry,
		OnEditingFinished: func() {
			config.SetConfItem(ci.Name, model.confItems[ci.Name].Entry.Text())
		},
	})
}

func RunMainWindow(config storage.ControllerConfig, configurations []ConfigurationLine, start func(context.Context, storage.ControllerConfig)) {
	model := &mainWindowModel{
		confItems: make(map[string]*lineEditHolder),
	}
	var doneFunc func()

	confList := GroupBox{
		Layout: Grid{Columns: 2},
		Title:  "Misc configuration",
	}

	for _, ci := range configurations {
		AddConfItem(model, &confList, ci, config)
	}

	mw := MainWindow{
		Title:    "The Buffbot",
		AssignTo: &model.mainWindow,
		MinSize:  Size{720, 200},
		Layout:   VBox{},
		Children: []Widget{
			GroupBox{
				Layout: Grid{Columns: 3},
				Title:  "EverQuest settings",
				Children: []Widget{
					Label{
						Text:          "EverQuest directory",
						TextAlignment: AlignFar,
					},
					LineEdit{
						AssignTo:          &model.dirEdit,
						OnEditingFinished: func() { onEqDirChanged(model, config) },
					},
					PushButton{
						Text:     "Browse...",
						AssignTo: &model.dirBrowse,
						OnClicked: func() {
							dialog := &walk.FileDialog{
								Title:    "Select EverQuest directory",
								FilePath: config.EverQuestDirectory(),
							}
							choose, err := dialog.ShowBrowseFolder(model.mainWindow)
							if err != nil {
								log.Println("Failed to show file dialog: %v", err)
								return
							}
							if choose {
								model.dirEdit.SetText(dialog.FilePath)
								onEqDirChanged(model, config)
							}
						},
					},
				},
			},
			confList,
			HSplitter{
				Children: []Widget{
					PushButton{
						Text:     "Start",
						Enabled:  false,
						AssignTo: &model.startButton,
						OnClicked: func() {
							if model.started {
								if doneFunc != nil {
									doneFunc()
									doneFunc = nil
								}
								model.started = false
								model.startButton.SetText("Start")
								model.shade()
							} else {
								model.started = true
								model.startButton.SetText("Stop")
								model.shade()
								var ctx context.Context
								ctx, doneFunc = context.WithCancel(context.Background())
								go func() {
									defer func() {
										if r := recover(); r != nil {
											log.Printf("Panic caught: %v", r)
										}
										model.mainWindow.Synchronize(func() {
											if doneFunc != nil {
												doneFunc()
												doneFunc = nil
											}
											model.started = false
											model.startButton.SetText("Start")
											model.shade()
										})
									}()
									start(ctx, config)
								}()
							}
						},
					},
				},
			},
		},
	}
	err := mw.Create()
	if err != nil {
		panic(err)
	}

	lv, _ := NewLogView(model.mainWindow)
	lv.PostAppendText("")
	log.SetOutput(lv)

	model.dirEdit.SetText(config.EverQuestDirectory())
	for _, ci := range configurations {
		model.confItems[ci.Name].Entry.SetText(config.GetConfItem(ci.Name))
	}
	onEqDirChanged(model, config)
	model.shade()

	model.mainWindow.Run()
}
