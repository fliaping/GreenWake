interface ParsedUserAgent {
  platform: string;
  browser: string;
}

export function parseUserAgent(userAgent: string): ParsedUserAgent {
  // 平台检测
  let platform = 'Unknown';
  if (userAgent.includes('Windows')) {
    platform = 'Windows';
  } else if (userAgent.includes('Macintosh')) {
    platform = 'macOS';
  } else if (userAgent.includes('Linux')) {
    platform = 'Linux';
  } else if (userAgent.includes('iPhone')) {
    platform = 'iOS';
  } else if (userAgent.includes('iPad')) {
    platform = 'iPadOS';
  } else if (userAgent.includes('Android')) {
    platform = 'Android';
  }

  // 浏览器检测 - 注意检测顺序很重要
  let browser = 'Unknown';
  if (userAgent.includes('Edg/')) {
    browser = 'Edge';
  } else if (userAgent.includes('Firefox/')) {
    browser = 'Firefox';
  } else if (userAgent.includes('Chrome/')) {
    browser = 'Chrome';
  } else if (userAgent.includes('Safari/') && !userAgent.includes('Chrome/')) {
    // Safari 需要特殊处理，因为 Chrome 也包含 Safari 字符串
    browser = 'Safari';
  } else if (userAgent.includes('OPR/') || userAgent.includes('Opera/')) {
    browser = 'Opera';
  }

  return { platform, browser };
}