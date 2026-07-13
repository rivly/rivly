import { LuMenu } from 'react-icons/lu'
import logo from '../assets/logo.png'
import { AccountMenu } from './AccountMenu'
import styles from './Topbar.module.css'

type Props = {
  onMenuToggle: () => void
}

export function Topbar({ onMenuToggle }: Props) {
  return (
    <header className={styles.topbar}>
      <button
        type="button"
        className={styles.burger}
        onClick={onMenuToggle}
        aria-label="Toggle menu"
      >
        <LuMenu />
      </button>
      <img
        className={styles.mobileLogo}
        src={logo}
        alt="Rivly"
        width={92}
        height={28}
      />
      <div className={styles.spacer} />
      <AccountMenu />
    </header>
  )
}
