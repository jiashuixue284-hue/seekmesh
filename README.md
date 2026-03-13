# SeekMesh：基于 EdgeVPN `pkg/node` 思路的 P2P 穿透 + 中继兜底 + Mesh 组网实现

本仓库实现了一套最小可运行的 `pkg/node` 方案，核心策略与 EdgeVPN 节点逻辑一致：

1. **优先直连（P2P）**：先尝试通过候选地址打洞/直拨。
2. **中继兜底**：直连失败则依次尝试配置的 Relay。
3. **Mesh 组网**：当目标节点既不直连也无中继会话时，从拓扑图选择下一跳进行多跳转发。

## 包结构

- `pkg/node/types.go`：节点抽象（`Session`、`Dialer`、`RelayPicker`、`Envelope`）
- `pkg/node/node.go`：连接策略与数据面转发（direct -> relay -> mesh）
- `pkg/node/topology.go`：拓扑维护与路由决策（Dijkstra 最短路径）
- `pkg/node/node_test.go`：三类关键测试（直连优先、Relay 回退、Mesh 多跳）

## 关键流程

### 1) 建连流程

`Node.Connect()` 的顺序：

- `DialDirect(peer)` 成功 -> 记录 direct session
- 失败 -> 遍历 `RelayPicker.Relays()`，尝试 `DialRelay(relay, peer)`
- 全部失败 -> 返回 `ErrNoRouteToPeer`

### 2) 发送流程

`Node.Send()` / `sendEnvelope()` 的顺序：

- 若存在目标节点 session -> 直接发（可能是 direct 或 relay）
- 否则查 `Topology.NextHop()`，按 mesh 路由发给下一跳
- 无下一跳 -> `ErrNoRouteToPeer`

### 3) Mesh 转发流程

`HandleIncoming()`：

- 若 `Destination == self` -> 投递给业务回调
- 否则 `HopCount++` 并检查 `HopLimit`
- 未超限则再次执行 `sendEnvelope()`，实现多跳转发

## 如何与真实网络对接

当前实现把网络能力抽象在 `Dialer` 与 `Session`：

- 你可以将 EdgeVPN 中已有的打洞、隧道、中继能力映射到：
  - `DialDirect(ctx, peer, candidates)`
  - `DialRelay(ctx, relay, peer)`
  - `Session.Send(ctx, env)`
- 并通过 gossip/控制面把链路关系写入 `InjectTopologyLink(a,b,kind)`，即可形成完整 mesh。

## 运行测试

```bash
go test ./...
```
