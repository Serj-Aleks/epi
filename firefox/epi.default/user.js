// EPI: настройки Firefox в образе
// не уходить в автономный режим, если при запуске сети ещё нет
user_pref("network.manage-offline-status", false);
user_pref("network.dnsCacheExpiration", 0);
// не предлагать восстановление сеанса после выключения кнопкой
user_pref("browser.sessionstore.resume_from_crash", false);
// без приветственных страниц и вопросов при первом запуске
user_pref("browser.aboutwelcome.enabled", false);
user_pref("trailhead.firstrun.didSeeAboutWelcome", true);
user_pref("browser.startup.homepage_override.mstone", "ignore");
user_pref("startup.homepage_welcome_url", "");
user_pref("startup.homepage_welcome_url.additional", "");
user_pref("browser.shell.checkDefaultBrowser", false);
user_pref("datareporting.policy.dataSubmissionPolicyBypassNotification", true);
