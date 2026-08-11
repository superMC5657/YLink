//! 用户端壳 —— 原则:Rust 侧尽量薄,只承担系统能力,业务逻辑全部在前端。
//! 见 docs/frontend/desktop-tauri.md §3-4。
//!
//! 插件注册:http / store / clipboard / opener / single-instance / autostart /
//! notification / process / window-state / os / deep-link(updater 需要发布配置,接入时启用)。
//! 桌面专属能力(托盘/菜单/单实例/自启/窗口状态记忆)以 `#[cfg(desktop)]` 隔离,
//! 移动端(Android/iOS)构建时自动排除,见 docs/frontend/desktop-tauri.md §9。

#[cfg(desktop)]
use tauri::Manager;

#[cfg(desktop)]
use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
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
        .invoke_handler(tauri::generate_handler![set_window_theme]);

    // 桌面专属插件:开机自启 / 窗口状态记忆 / 深链接(移动端无对应概念,见 desktop-tauri.md §3)
    #[cfg(desktop)]
    let builder = builder
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            None,
        ))
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_deep_link::init());

    // 单实例:第二个实例启动时聚焦已有窗口并转发参数(deep-link 接入时转发 URL)
    #[cfg(desktop)]
    let builder = builder.plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
        if let Some(window) = app.get_webview_window("main") {
            let _ = window.set_focus();
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
                let _tray = TrayIconBuilder::new()
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
