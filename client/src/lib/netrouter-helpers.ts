/**
 * Design reminder — utility logic supports compact native interactions and
 * clear validation feedback rather than hidden state or ambiguous errors.
 */
export function isValidConnectionTarget(value: string): boolean {
  const candidate = value.trim();
  const macPattern = /^([0-9A-F]{2}:){5}[0-9A-F]{2}$/i;
  const ipv4Pattern = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;
  return macPattern.test(candidate) || ipv4Pattern.test(candidate);
}

export function connectionError(target: string, username: string): string | null {
  if (!isValidConnectionTarget(target)) return "Invalid IP address or MAC address.";
  if (!username.trim()) return "Authentication failed: enter a username.";
  return null;
}
