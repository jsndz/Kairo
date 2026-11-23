export function NumberField({
  label,
  value,
  change,
  max,
  min = 0,
  step = 1,
  unit = 's',
}: {
  label: string
  value: number
  change: (value: number) => void
  max: number
  min?: number
  step?: number
  unit?: string
}) {
  return (
    <label className="number-field">
      <span>{label}</span>
      <span className="number-input">
        <input
          key={value}
          type="number"
          min={min}
          max={max}
          step={step}
          defaultValue={+value.toFixed(3)}
          onBlur={event => {
            const next = event.currentTarget.valueAsNumber
            if (Number.isFinite(next)) change(Math.max(min, Math.min(max, next)))
            event.currentTarget.value = String(+value.toFixed(3))
          }}
          onKeyDown={event => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
        />
        <small>{unit}</small>
      </span>
    </label>
  )
}
