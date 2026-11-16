using System;
using System.Diagnostics;
using System.Drawing;
using System.Drawing.Drawing2D;
using System.IO;
using System.Text;
using System.Web.Script.Serialization;
using System.Windows.Forms;

public sealed class GameCell
{
    public int i { get; set; }
    public string t { get; set; }
    public int w { get; set; }
    public int f { get; set; }
    public int b { get; set; }
}

public sealed class GameFrame
{
    public int width { get; set; }
    public int height { get; set; }
    public int background { get; set; }
    public GameCell[] cells { get; set; }
}

internal sealed class GameWindow : Form
{
    private const int CellWidth = 14;
    private const int CellHeight = 28;
    private const int BoardPadding = 20;
    private const int Background = 0x101222;

    private readonly Process engine;
    private readonly Timer timer = new Timer { Interval = 16 };
    private readonly JavaScriptSerializer serializer = new JavaScriptSerializer();
    private readonly Font font = new Font("맑은 고딕", 18, FontStyle.Regular, GraphicsUnit.Pixel);
    private readonly Font wideFont = new Font("맑은 고딕", 22, FontStyle.Regular, GraphicsUnit.Pixel);
    private readonly StringFormat textFormat = new StringFormat(StringFormatFlags.NoClip | StringFormatFlags.NoWrap)
    {
        Alignment = StringAlignment.Center,
        LineAlignment = StringAlignment.Center
    };
    private volatile GameFrame latestFrame;
    private GameFrame frame;

    private GameWindow()
    {
        Text = "픽셀 테트리스";
        ClientSize = new Size(47 * CellWidth + 2 * BoardPadding, 24 * CellHeight + 2 * BoardPadding);
        MinimumSize = new Size(490, 530);
        StartPosition = FormStartPosition.CenterScreen;
        DoubleBuffered = true;
        ResizeRedraw = true;

        engine = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "engine", "pixel-tetris.exe"),
                Arguments = "--window",
                UseShellExecute = false,
                CreateNoWindow = true,
                RedirectStandardInput = true,
                RedirectStandardOutput = true,
                StandardOutputEncoding = Encoding.UTF8
            }
        };
        engine.OutputDataReceived += ReadFrame;
        engine.Start();
        engine.BeginOutputReadLine();
        timer.Tick += RefreshFrame;
        timer.Start();
    }

    [STAThread]
    private static void Main()
    {
        Application.EnableVisualStyles();
        try
        {
            using (var window = new GameWindow())
                Application.Run(window);
        }
        catch (System.ComponentModel.Win32Exception)
        {
            MessageBox.Show("게임 엔진을 시작하지 못했습니다. 압축을 모두 푼 뒤 다시 실행해 주세요.",
                "픽셀 테트리스", MessageBoxButtons.OK, MessageBoxIcon.Error);
        }
    }

    private void ReadFrame(object sender, DataReceivedEventArgs args)
    {
        if (args.Data != null)
            latestFrame = serializer.Deserialize<GameFrame>(args.Data);
    }

    private void RefreshFrame(object sender, EventArgs args)
    {
        if (engine.HasExited)
        {
            if (engine.ExitCode != 0)
            {
                timer.Stop();
                MessageBox.Show(this, "게임 엔진이 예기치 않게 종료되었습니다.", Text,
                    MessageBoxButtons.OK, MessageBoxIcon.Error);
            }
            Close();
            return;
        }
        var next = latestFrame;
        if (frame != next)
        {
            frame = next;
            Invalidate();
        }
    }

    private void Send(string input)
    {
        try
        {
            engine.StandardInput.WriteLine(input);
        }
        catch (IOException)
        {
            // 엔진이 입력을 받기 전에 종료된 경우 다음 화면 갱신에서 창을 닫는다.
        }
    }

    protected override bool ProcessCmdKey(ref Message message, Keys key)
    {
        string input;
        switch (key)
        {
            case Keys.Enter: input = "enter"; break;
            case Keys.Escape: input = "escape"; break;
            case Keys.Left: input = "left"; break;
            case Keys.Right: input = "right"; break;
            case Keys.Up: input = "up"; break;
            case Keys.Down: input = "down"; break;
            case Keys.Space: input = " "; break;
            case Keys.A: case Keys.D: case Keys.W: case Keys.S:
            case Keys.P: case Keys.R: case Keys.M: case Keys.Q:
                input = ((char)key).ToString().ToLowerInvariant();
                break;
            default: return base.ProcessCmdKey(ref message, key);
        }
        Send(input);
        return true;
    }

    private float GetPlacement(out PointF origin)
    {
        var width = frame.width * CellWidth;
        var height = frame.height * CellHeight;
        var scale = Math.Min((ClientSize.Width - 2f * BoardPadding) / width,
            (ClientSize.Height - 2f * BoardPadding) / height);
        origin = new PointF((ClientSize.Width - width * scale) / 2,
            (ClientSize.Height - height * scale) / 2);
        return scale;
    }

    protected override void OnMouseDown(MouseEventArgs args)
    {
        base.OnMouseDown(args);
        if (args.Button != MouseButtons.Left || frame == null)
            return;

        PointF origin;
        var scale = GetPlacement(out origin);
        var x = (int)Math.Floor((args.X - origin.X) / (CellWidth * scale));
        var y = (int)Math.Floor((args.Y - origin.Y) / (CellHeight * scale));
        if (x >= 0 && x < frame.width && y >= 0 && y < frame.height)
            Send(string.Format("click {0} {1}", x, y));
    }

    private static Color FromRgb(int value)
    {
        return Color.FromArgb(unchecked((int)0xff000000) | (value < 0 ? 0xe4dbfb : value));
    }

    protected override void OnPaintBackground(PaintEventArgs args)
    {
        args.Graphics.Clear(FromRgb(frame == null ? Background : frame.background));
    }

    protected override void OnPaint(PaintEventArgs args)
    {
        if (frame == null)
            return;

        PointF origin;
        var scale = GetPlacement(out origin);
        var graphics = args.Graphics;
        graphics.TranslateTransform(origin.X, origin.Y);
        graphics.ScaleTransform(scale, scale);
        graphics.PixelOffsetMode = PixelOffsetMode.Half;
        using (var brush = new SolidBrush(Color.Black))
        {
            foreach (var cell in frame.cells)
            {
                if (cell.b == frame.background)
                    continue;
                brush.Color = FromRgb(cell.b);
                graphics.FillRectangle(brush, CellBounds(cell));
            }
            foreach (var cell in frame.cells)
            {
                if (cell.t == null)
                    continue;
                brush.Color = FromRgb(cell.f);
                DrawCell(graphics, brush, cell);
            }
        }
    }

    private RectangleF CellBounds(GameCell cell)
    {
        return new RectangleF(cell.i % frame.width * CellWidth, cell.i / frame.width * CellHeight,
            cell.w * CellWidth, CellHeight);
    }

    private void DrawCell(Graphics graphics, SolidBrush brush, GameCell cell)
    {
        var rect = CellBounds(cell);
        var midX = rect.X + rect.Width / 2;
        var midY = rect.Y + rect.Height / 2;
        switch (cell.t)
        {
            case "▌": graphics.FillRectangle(brush, rect.X, rect.Y, rect.Width / 2, rect.Height); break;
            case "▀": graphics.FillRectangle(brush, rect.X, rect.Y, rect.Width, rect.Height / 2); break;
            case "▄": graphics.FillRectangle(brush, rect.X, midY, rect.Width, rect.Height / 2); break;
            case "·": graphics.FillRectangle(brush, midX - 1, midY - 1, 2, 2); break;
            case "─": case "│": case "┌": case "┐": case "└": case "┘":
                var left = "─┐┘".Contains(cell.t) ? rect.Left : midX;
                var right = "─┌└".Contains(cell.t) ? rect.Right : midX;
                var top = "│└┘".Contains(cell.t) ? rect.Top : midY;
                var bottom = "│┌┐".Contains(cell.t) ? rect.Bottom : midY;
                if (left < right)
                    graphics.FillRectangle(brush, left - 0.5f, midY - 0.5f, right - left + 1, 1);
                if (top < bottom)
                    graphics.FillRectangle(brush, midX - 0.5f, top - 0.5f, 1, bottom - top + 1);
                break;
            default:
                graphics.DrawString(cell.t, cell.w > 1 ? wideFont : font, brush, rect, textFormat);
                break;
        }
    }

    protected override void OnFormClosed(FormClosedEventArgs args)
    {
        timer.Dispose();
        if (!engine.HasExited)
        {
            Send("escape");
            if (!engine.WaitForExit(1000))
            {
                try { engine.Kill(); }
                catch (InvalidOperationException) { }
            }
        }
        engine.Dispose();
        textFormat.Dispose();
        wideFont.Dispose();
        font.Dispose();
        base.OnFormClosed(args);
    }
}
