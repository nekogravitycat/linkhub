import { useWebHaptics } from "web-haptics/vue"

export function useAppHaptics() {
  const { trigger } = useWebHaptics()

  const hapticError = () =>
    trigger([{ duration: 40 }, { delay: 40, duration: 40 }, { delay: 40, duration: 40 }], {
      intensity: 0.9,
    })

  const hapticSuccess = () => trigger([{ duration: 30 }, { delay: 60, duration: 40, intensity: 1 }])

  const hapticNudge = () =>
    trigger([
      { duration: 80, intensity: 0.8 },
      { delay: 80, duration: 50, intensity: 0.3 },
    ])

  return {
    hapticError,
    hapticSuccess,
    hapticNudge,
  }
}
