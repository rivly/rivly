import { LuInfo } from 'react-icons/lu'
import { Tooltip } from './Tooltip'
import styles from './LimitedBadge.module.css'

export function LimitedBadge() {
  return (
    <Tooltip content="This stack was created outside Rivly, so control over it is limited.">
      <span className={styles.badge}>
        <LuInfo />
        Limited
      </span>
    </Tooltip>
  )
}
