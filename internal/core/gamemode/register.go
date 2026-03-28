// register.go — 自动注册所有内置游戏模式。
package gamemode

func init() {
	Register(NewCampaignMode())
	Register(NewEndlessMode())
	Register(NewTimedMode())
	Register(NewBossRushMode())
	Register(NewChallengeMode())
	Register(NewTestMode())
}
