const environmentNamePattern = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/
const applicationVersionPattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/

export function isValidEnvironmentName(value: string) {
  return value.length <= 63 && environmentNamePattern.test(value)
}

export function isValidApplicationVersion(value: string) {
  return value === '' || applicationVersionPattern.test(value)
}
