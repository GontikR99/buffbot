package storage

import (
	"log"
)

type BoltholdBackedConfig struct {
}

type bhConfigEntry struct {
	Data []byte
}

const (
	eqDirKey    = "eqDir"
	charNameKey = "charName"
	varConfKey  = "varConf:"
)

func (bhc *BoltholdBackedConfig) EverQuestDirectory() string {
	value := &bhConfigEntry{}
	err := database.Get(eqDirKey, value)
	if err == nil {
		return string(value.Data)
	} else {
		return "C:\\Users\\Public\\Daybreak Game Company\\Installed Games\\EverQuest"
	}
}

func (bhc *BoltholdBackedConfig) SelectedCharacter() string {
	value := &bhConfigEntry{}
	err := database.Get(charNameKey, value)
	if err == nil {
		return string(value.Data)
	} else {
		return ""
	}
}

func (bhc *BoltholdBackedConfig) SetEverQuestDirectory(eqDir string) {
	err := database.Upsert(eqDirKey, &bhConfigEntry{[]byte(eqDir)})
	if err != nil {
		log.Println(err)
	}
}

func (bhc *BoltholdBackedConfig) SetSelectedCharacter(charName string) {
	err := database.Upsert(charNameKey, &bhConfigEntry{[]byte(charName)})
	if err != nil {
		log.Println(err)
	}
}

func (bhc *BoltholdBackedConfig) GetConfItem(name string) string {
	value := &bhConfigEntry{}
	err := database.Get(varConfKey+name, value)
	if err == nil {
		return string(value.Data)
	} else {
		return ""
	}
}

func (bhc *BoltholdBackedConfig) HasConfItem(name string) bool {
	value := &bhConfigEntry{}
	err := database.Get(varConfKey+name, value)
	return err==nil
}


func (bhc *BoltholdBackedConfig) SetConfItem(name, value string) {
	err := database.Upsert(varConfKey+name, &bhConfigEntry{[]byte(value)})
	if err != nil {
		log.Println(err)
	}
}
