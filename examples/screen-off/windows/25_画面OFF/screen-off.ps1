# モニターの電源だけを切る(PC は動いたままで、ゲームやサーバーは止まらない)。
# マウスやキーに触ると点く。
# SendMessage ではなく PostMessage にするのは、全ウィンドウ宛て(HWND_BROADCAST)だと、
# 応答しないウィンドウが 1 つあるだけで SendMessage が返らなくなるため。
Add-Type -Namespace Kido -Name Monitor -MemberDefinition @'
[DllImport("user32.dll")]
public static extern bool PostMessage(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);
'@
$HWND_BROADCAST = [IntPtr]0xffff
$WM_SYSCOMMAND = 0x0112
$SC_MONITORPOWER = [IntPtr]0xF170
$MONITOR_OFF = [IntPtr]2
[Kido.Monitor]::PostMessage($HWND_BROADCAST, $WM_SYSCOMMAND, $SC_MONITORPOWER, $MONITOR_OFF) | Out-Null
