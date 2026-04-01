我建议先去调整系统代码，然后再更新/Users/yushion/Games/defense2/docs/config-audit-report.md。我的手动审核如下：炮塔没有升级了，去掉基础升级费用，删掉InterestRate/InterestCap，SellRefundRatio设定为0.7。我们
当前实现应该就不需要towers.json了。也没有升级费用。护盾机制我们已经删除。敌人无参数的buff都可以删除。每波数量公式按照5 +                                                                                       
wave，CC类型没有定身。我们没有魔法伤害，应该也没有真实伤害吧（管线有处理这个吗？）。波次难度缩放中生命值，JSON 公式/Go公式，采用JSON公式。出怪计数按照Go                                                       
5+wave。系统里面armorDivisor，护甲除数也可以删除，我们不需要护甲。dotTickInterval.poison这个按照 Go 统一 DotTickInterval=0.5s 。然后JSON maxDamageAmplification=3 vs Go                                        
MaxDamageAmplify=0.5这个就按照Go的。护甲系统删除。 

---