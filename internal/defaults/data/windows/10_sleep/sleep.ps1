# スリープにする。
# rundll32 powrprof.dll,SetSuspendState は、休止状態が有効な PC では
# スリープではなく休止状態に入ってしまうので、第 1 引数で明示する。
# 戻り値は 1 バイトの BOOLEAN。.NET の Application.SetSuspendState や 4 バイトで読むと、
# 権限が足りずに失敗しても真に見える。
Add-Type -Namespace Kido -Name Power -MemberDefinition @'
[DllImport("powrprof.dll", SetLastError = true)]
[return: MarshalAs(UnmanagedType.I1)]
public static extern bool SetSuspendState(
    [MarshalAs(UnmanagedType.I1)] bool hibernate,
    [MarshalAs(UnmanagedType.I1)] bool force,
    [MarshalAs(UnmanagedType.I1)] bool wakeupEventsDisabled);
'@
if (-not [Kido.Power]::SetSuspendState($false, $false, $false)) {
    [Console]::Error.WriteLine("SetSuspendState failed: " + [Runtime.InteropServices.Marshal]::GetLastWin32Error())
    exit 1
}
