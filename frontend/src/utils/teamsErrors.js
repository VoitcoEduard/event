const teamErrorKeys = {
  'Assign at least one participant before starting': 'teams_module.errors.assign_participant_before_start'
}

export function localizeTeamsError(message, translate) {
  const key = teamErrorKeys[message]
  return key ? translate(key) : message
}
