//! 用户端壳 —— 原则:Rust 侧尽量薄,只承担系统能力,业务逻辑全部在前端。
//! 见 docs/frontend/desktop-tauri.md §3-4。
//!
//! 插件注册:http / store / clipboard / opener / single-instance / autostart /
//! notification / process / window-state / os / deep-link / updater(签名与端点见 tauri.conf.json plugins.updater)。
//! 桌面专属能力(托盘/菜单/单实例/自启/窗口状态记忆)以 `#[cfg(desktop)]` 隔离,
//! 移动端(Android/iOS)构建时自动排除,见 docs/frontend/desktop-tauri.md §9。

#[cfg(desktop)]
use tauri::Manager;

#[cfg(desktop)]
use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    Emitter,
};

/// 主题切换监听:前端 `theme-changed` 事件 → 设置窗口标题栏亮暗。
/// 见 docs/frontend/desktop-tauri.md §4「主题跟随」。
/// 窗口尺寸/位置由 window-state 插件自动持久化,无需手动调用。
#[tauri::command]
fn set_window_theme(window: tauri::Window, dark: bool) {
    let _ = window.set_theme(if dark {
        Some(tauri::Theme::Dark)
    } else {
        Some(tauri::Theme::Light)
    });
}

/// 移动端入口:Android 由 MainActivity 经 JNI 调用;桌面入口在 main.rs。
#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .invoke_handler(tauri::generate_handler![set_window_theme]);

    // 深链接:桌面 + 移动端都注册(Android 经 manifest intent-filter 接收 ylink://,
    // iOS 经 Info.plist CFBundleURLTypes;桌面 Windows 注册表协议由插件管理)。
    // 移动端 scheme 见 tauri.conf.json plugins.deep-link.mobile。
    let builder = builder.plugin(tauri_plugin_deep_link::init());

    // 桌面专属插件:开机自启 / 窗口状态记忆
    #[cfg(desktop)]
    let builder = builder
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            None,
        ))
        .plugin(tauri_plugin_window_state::Builder::default().build());

    // 单实例:第二个实例启动时聚焦已有窗口,并把 argv 中的深链接 URL 转发给已有实例。
    // 事件名与 deep-link 插件一致(`deep-link://new-url`,payload 为 URL 数组),
    // 前端 onOpenUrl(见 utils/platform.ts onDeepLink)无需改动即可收到,
    // 实现「重复启动/协议唤起时定位到已有窗口对应页面」(desktop-tauri.md §4)。
    #[cfg(desktop)]
    let builder = builder.plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
        if let Some(window) = app.get_webview_window("main") {
            let _ = window.set_focus();
            let urls: Vec<String> = argv
                .iter()
                .skip(1) // argv[0] 为可执行文件路径
                .filter(|a| a.starts_with("ylink://"))
                .cloned()
                .collect();
            if !urls.is_empty() {
                let _ = window.emit("deep-link://new-url", urls);
            }
        }
        log::info!("second instance argv: {:?}", argv);
    }));

    builder
        .setup(|app| {
            #[cfg(not(desktop))]
            let _ = app;

            // 托盘:显示主窗口 / 退出(仅桌面;移动端无托盘,检查更新在 updater 接入后追加)
            #[cfg(desktop)]
            {
                let show_i = MenuItem::with_id(app, "show", "显示主窗口", true, None::<&str>)?;
                let quit_i = MenuItem::with_id(app, "quit", "退出", true, None::<&str>)?;
                let menu = Menu::with_items(app, &[&show_i, &quit_i])?;
                let _tray = TrayIconBuilder::with_id("main-tray")
                    .icon(app.default_window_icon().unwrap().clone())
                    .menu(&menu)
                    .show_menu_on_left_click(false)
                    .on_menu_event(|app, event| match event.id.as_ref() {
                        "show" => {
                            if let Some(window) = app.get_webview_window("main") {
                                let _ = window.show();
                                let _ = window.set_focus();
                            }
                        }
                        "quit" => app.exit(0),
                        _ => {}
                    })
                    .build(app)?;
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
