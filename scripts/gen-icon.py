"""生成 YLink 品牌图标源图(1024x1024 PNG,纯 Python 无依赖)。
主色 #6558F5 背景 + 白色闪电(与 Web 端 favicon 一致)。"""
import struct
import zlib


def make_png(width, height, pixel_fn):
    rows = []
    for y in range(height):
        row = bytearray([0])  # filter type 0
        for x in range(width):
            row.extend(pixel_fn(x, y))
        rows.append(bytes(row))
    raw = b"".join(rows)

    def chunk(ctype, data):
        c = ctype + data
        return struct.pack(">I", len(data)) + c + struct.pack(">I", zlib.crc32(c))

    ihdr = struct.pack(">IIBBBBB", width, height, 8, 6, 0, 0, 0)
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )


# 闪电多边形(规范化坐标 0-1,中心放大)
BOLT = [(0.56, 0.16), (0.40, 0.52), (0.50, 0.52), (0.42, 0.86), (0.66, 0.46), (0.55, 0.46), (0.64, 0.16)]


def point_in_poly(x, y, poly):
    inside = False
    n = len(poly)
    for i in range(n):
        x1, y1 = poly[i]
        x2, y2 = poly[(i + 1) % n]
        if (y1 > y) != (y2 > y) and x < (x2 - x1) * (y - y1) / (y2 - y1) + x1:
            inside = not inside
    return inside


def pixel(x, y):
    # 圆角背景(圆角半径 ~180)
    r = 180
    w, h = 1024, 1024
    def rounded(cx, cy):
        return (
            (r <= cx < w - r or r <= cy < h - r)
            or ((cx - r) ** 2 + (cy - r) ** 2 <= r**2)
            or ((cx - (w - r)) ** 2 + (cy - r) ** 2 <= r**2)
            or ((cx - r) ** 2 + (cy - (h - r)) ** 2 <= r**2)
            or ((cx - (w - r)) ** 2 + (cy - (h - r)) ** 2 <= r**2)
        )
    if not rounded(x + 0.5, y + 0.5):
        return bytes((0, 0, 0, 0))
    # 闪电(0.55 缩放居中)
    nx = x / w
    ny = y / h
    lx = (nx - 0.5) / 0.55 + 0.5
    ly = (ny - 0.5) / 0.55 + 0.5
    if point_in_poly(lx, ly, BOLT):
        return bytes((255, 255, 255, 255))
    return bytes((0x65, 0x58, 0xF5, 255))


with open("app-icon.png", "wb") as f:
    f.write(make_png(1024, 1024, pixel))
print("app-icon.png generated")
