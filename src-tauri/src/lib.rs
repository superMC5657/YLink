//! 用户端桌面壳 —— 原则:Rust 侧尽量薄,只承担系统能力,业务逻辑全部在前端。
//! 见 docs/frontend/desktop-tauri.md §3-4。
//!
//! 插件注册:http / store / clipboard / opener / single-instance / autostart /
//! notification / process / window-state / os(deep-link 与 updater 需要发布配置,接入时启用)。

use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    Manager,
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

pub fn run() {
    let mut builder = tauri::Builder::default()
        .plugin(tauri_plugin_http::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_clipboard_manager::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            None,
        ))
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_window_state::Builder::default().build())
        .plugin(tauri_plugin_os::init())
        .invoke_handler(tauri::generate_handler![set_window_theme]);

    // 单实例:第二个实例启动时聚焦已有窗口并转发参数(deep-link 接入时转发 URL)
    builder = builder.plugin(tauri_plugin_single_instance::init(|app, argv, _cwd| {
        if let Some(window) = app.get_webview_window("main") {
            let _ = window.set_focus();
        }
        log::info!("second instance argv: {:?}", argv);
    }));

    builder
        .setup(|app| {
            // 托盘:显示主窗口 / 退出(检查更新在 updater 接入后追加)
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
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
