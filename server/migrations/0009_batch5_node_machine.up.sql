-- 0009_batch5_node_machine.up.sql
-- F17 节点机器管理:
--   预置 node 站点设置项: install_script_url, install_command_template (中台可直接修改)

INSERT INTO settings ("key", value)
VALUES (
    'node',
    '{"install_script_url":"https://raw.githubusercontent.com/superMC5657/YLink/main/deploy/install-agent.sh","install_command_template":"curl -fsSL {{script_url}} | bash -s -- --endpoint {{endpoint}} --key {{node_key}}"}'
)
ON CONFLICT ("key") DO NOTHING;
