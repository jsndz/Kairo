import { Aperture, ArrowUpRight, GitFork } from 'lucide-react'
import { projectsHref, repositoryUrl } from '../config/navigation'

export function Navbar() {
  return (
    <header className="landing-nav">
      <nav className="landing-nav-inner" aria-label="Main navigation">
        <a className="landing-wordmark" href={import.meta.env.BASE_URL}>
          <Aperture size={23} aria-hidden="true" />
          Kairo
        </a>
        <div className="landing-nav-actions">
          {repositoryUrl && (
            <a className="landing-repo-link" href={repositoryUrl} aria-label="Kairo on GitHub">
              <GitFork size={17} aria-hidden="true" />
              <span className="hidden sm:inline">GitHub</span>
            </a>
          )}
          <a className="landing-button" href={projectsHref}>
            Launch Editor <ArrowUpRight size={15} aria-hidden="true" />
          </a>
        </div>
      </nav>
    </header>
  )
}
