// Date helpers mirroring apps/web/src/utils/date.ts.

const SHANGHAI_TZ = 'Asia/Shanghai';

// formatInTimeZone equivalent supporting the tokens used by the app:
// yyyy MM dd HH mm ss plus single-letter M d H m s (unpadded).
export function formatInTimeZone(date: Date, timeZone: string, formatStr: string): string {
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    }).formatToParts(date);
    const map: Record<string, string> = {};
    for (const p of parts) map[p.type] = p.value;
    let hour = map.hour;
    if (hour === '24') hour = '00';
    const tokens: Record<string, string> = {
      yyyy: map.year,
      MM: map.month,
      dd: map.day,
      HH: hour,
      mm: map.minute,
      ss: map.second,
      M: String(parseInt(map.month, 10)),
      d: String(parseInt(map.day, 10)),
      H: String(parseInt(hour, 10)),
      m: String(parseInt(map.minute, 10)),
      s: String(parseInt(map.second, 10))
    };
    let out = '';
    let i = 0;
    while (i < formatStr.length) {
      // try 4-char token, then 2-char, then 1-char
      const four = formatStr.slice(i, i + 4);
      if (four === 'yyyy') {
        out += tokens['yyyy'];
        i += 4;
        continue;
      }
      const two = formatStr.slice(i, i + 2);
      if (tokens[two] !== undefined) {
        out += tokens[two];
        i += 2;
        continue;
      }
      const one = formatStr[i];
      if (tokens[one] !== undefined) {
        out += tokens[one];
        i += 1;
        continue;
      }
      out += one;
      i += 1;
    }
    return out;
  } catch {
    return '';
  }
}

export function formatChinaTime(date: Date, formatStr = 'yyyy-MM-dd HH:mm') {
  if (!date || Number.isNaN(date.getTime())) return '';
  return formatInTimeZone(date, SHANGHAI_TZ, formatStr);
}

export function getWeekday(dateString: string): string {
  try {
    const [year, month, day] = dateString.split('-').map(Number);
    const date = new Date(Date.UTC(year, month - 1, day));
    if (Number.isNaN(date.getTime())) return '';
    const weekdays = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六'];
    return weekdays[date.getUTCDay()];
  } catch {
    return '';
  }
}
