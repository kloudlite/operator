local dap = require("dap")

dap.configurations.go = {
  {
    type = "go",
    name = "Debug operator project",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/project",
    args = { "--dev", "--serverHost", "localhost:8080" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/project" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug platform operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/cmd/platform-operator",
    args = { "--dev", "--serverHost", "localhost:8080" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/cmd/platform-operator" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug agent operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/cmd/agent-operator",
    args = { "--dev", "--serverHost", "localhost:8081" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/cmd/agent-operator" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug app-n-lambda",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/app-n-lambda",
    -- args = { "--dev", "" },
    args = { "--dev", "--serverHost", "localhost:8080" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/app-n-lambda" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug account",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/account",
    -- args = { "--dev", "" },
    args = { "--dev", "--serverHost", "localhost:8080" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/account" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug artifacts-harbor",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/artifacts-harbor",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/artifacts-harbor" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug routers",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/routers",
    args = { "--dev", "--serverHost", "localhost:8080" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/routers" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug cluster-setup",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/cluster-setup",
    args = { "--dev" },
    -- console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/cluster-setup" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug resource-watcher",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/resource-watcher",
    -- args = { "--dev", "--serverHost", "localhost:8080", "--running-on-platform" },
    args = { "--dev", "--serverHost", "localhost:8081" },
    -- console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/resource-watcher" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug agent",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/agent",
    args = { "--dev" },
    -- console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/agent" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug byoc-helm-status-watcher",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/byoc-helm-status-watcher",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/byoc-helm-status-watcher" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug byoc-operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/byoc-operator",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/byoc-operator" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug byoc-client-operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/byoc-client-operator",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/byoc-client-operator" .. "/.secrets/env",
    },
  },
  {
    type = "go_test",
    name = "[Debug] Test app-n-lambda",
    request = "remote",
    mode = "test",
    program = vim.g.nxt.project_root_dir .. "/operators/app-n-lambda/internal/controllers/app/control",
    env = {
      PROJECT_ROOT = vim.g.nxt.project_root_dir,
    },
  },
  {
    type = "go",
    name = "Debug msvc-mongo",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/msvc-mongo",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/msvc-mongo" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug msvc-redis",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/msvc-redis",
    args = { "--dev" },
    console = "externalTerminal",
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/msvc-redis" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug clusters",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/clusters",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/clusters" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug msvc-n-mres",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/msvc-n-mres",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/msvc-n-mres" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug msvc-redpanda",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/msvc-redpanda",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    env = {
      RECONCILE_PERIOD = "30s",
      MAX_CONCURRENT_RECONCILES = "1",
    },
  },
  {
    type = "go",
    name = "Debug helm-charts-controller",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/helm-charts",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/helm-charts" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug wireguard-operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/wireguard",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/wireguard" .. "/.secrets/env",
    },
  },
  {
    type = "go",
    name = "Debug nodepool operator",
    request = "launch",
    program = vim.g.nxt.project_root_dir .. "/operators/nodepool",
    args = { "--dev" },
    console = "externalTerminal",
    -- externalTerminal = true,
    envFile = {
      vim.g.nxt.project_root_dir .. "/operators/nodepool" .. "/.secrets/env",
    },
  },
}
