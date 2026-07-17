import { useMutation, useQueryClient } from '@tanstack/react-query'
import { type FormEvent, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { createEnvironment } from '../api/environments'
import { isValidApplicationVersion, isValidEnvironmentName } from './createEnvironmentValidation'

type WorkloadProfile = 'healthy' | 'unhealthy' | 'custom'

const healthyDemoImage = 'envpilot/demo-service:healthy'

export function CreateEnvironmentPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [profile, setProfile] = useState<WorkloadProfile>('healthy')
  const [image, setImage] = useState('')
  const [containerPort, setContainerPort] = useState('8080')
  const [applicationVersion, setApplicationVersion] = useState('')
  const mutation = useMutation({
    mutationFn: createEnvironment,
    onSuccess: async (environment) => {
      await queryClient.invalidateQueries({ queryKey: ['environments'] })
      navigate(`/environments/${environment.id}`)
    },
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (mutation.isPending) return
    const normalizedName = name.trim()
    const normalizedVersion = applicationVersion.trim()
    const normalizedImage = profile === 'custom' ? image.trim() : healthyDemoImage
    const parsedContainerPort = Number(containerPort)
    if (!isValidEnvironmentName(normalizedName) || !isValidApplicationVersion(normalizedVersion)) return
    if (!normalizedImage || !Number.isInteger(parsedContainerPort) || parsedContainerPort < 1 || parsedContainerPort > 65535) return
    mutation.mutate({
      name: normalizedName,
      image: normalizedImage,
      containerPort: parsedContainerPort,
      simulateFailure: profile === 'unhealthy',
      ...(normalizedVersion ? { applicationVersion: normalizedVersion } : {}),
    })
  }

  return (
    <section>
      <div className="page-heading">
        <div>
          <p className="eyebrow">New environment</p>
          <h1>Create an ephemeral environment</h1>
          <p>Launch a constrained container workload and follow each provisioning step.</p>
        </div>
      </div>
      <form className="operator-form" onSubmit={submit}>
        <div className="form-section">
          <div className="form-section-heading">
            <span>01</span><div><h2>Environment identity</h2><p>Use a short name that operators can recognize.</p></div>
          </div>
          <label className="field-label" htmlFor="environment-name">Environment name</label>
          <input
            id="environment-name"
            name="name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="feature-payment"
            pattern="[a-z0-9](?:[a-z0-9-]*[a-z0-9])?"
            maxLength={63}
            autoComplete="off"
            required
          />
          <p className="field-help">Lowercase letters, numbers, and hyphens. Maximum 63 characters.</p>
        </div>

        <div className="form-section">
          <div className="form-section-heading">
            <span>02</span><div><h2>Workload profile</h2><p>Choose a demo profile or provide another Docker image.</p></div>
          </div>
          <div className="profile-grid">
            <ProfileOption
              value="healthy" selected={profile === 'healthy'} onSelect={setProfile}
              title="Healthy demo service" description="Expected to provision and become ready."
            />
            <ProfileOption
              value="unhealthy" selected={profile === 'unhealthy'} onSelect={setProfile}
              title="Simulated health failure" description="Returns 503 so the workflow failure path is visible."
            />
            <ProfileOption
              value="custom" selected={profile === 'custom'} onSelect={setProfile}
              title="Custom Docker image" description="Run another image available to the Docker Engine."
            />
          </div>
          {profile === 'custom' ? (
            <div className="custom-image-fields form-grid">
              <div>
                <label className="field-label" htmlFor="container-image">Container image</label>
                <input
                  id="container-image"
                  value={image}
                  onChange={(event) => setImage(event.target.value)}
                  placeholder="nginx:latest"
                  maxLength={255}
                  autoComplete="off"
                  required
                />
                <p className="field-help">Use an image name already available to the Docker Engine.</p>
              </div>
              <div>
                <label className="field-label" htmlFor="container-port">Container port</label>
                <input
                  id="container-port"
                  type="number"
                  value={containerPort}
                  onChange={(event) => setContainerPort(event.target.value)}
                  min="1"
                  max="65535"
                  required
                />
                <p className="field-help">The HTTP port exposed by the image.</p>
              </div>
            </div>
          ) : null}
        </div>

        <div className="form-section form-grid">
          {profile !== 'custom' ? (
            <div>
              <label className="field-label" htmlFor="demo-container-port">Container port</label>
              <input id="demo-container-port" value="8080" readOnly aria-readonly="true" />
              <p className="field-help">Defined by the selected demo workload.</p>
            </div>
          ) : null}
          <div>
            <label className="field-label" htmlFor="application-version">Application version <span>Optional</span></label>
            <input
              id="application-version"
              value={applicationVersion}
              onChange={(event) => setApplicationVersion(event.target.value)}
              placeholder="1.4.0-rc.1"
              pattern="[A-Za-z0-9][A-Za-z0-9._-]{0,63}"
              maxLength={64}
            />
            <p className="field-help">Passed to the container as APP_VERSION.</p>
          </div>
        </div>

        {mutation.isError ? <SubmissionError error={mutation.error} /> : null}

        <div className="form-actions">
          <button className="secondary-button" type="button" onClick={() => navigate('/')} disabled={mutation.isPending}>Cancel</button>
          <button type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? 'Creating environment…' : 'Create environment'}
          </button>
        </div>
      </form>
    </section>
  )
}

function ProfileOption({
  value, selected, onSelect, title, description,
}: {
  value: WorkloadProfile
  selected: boolean
  onSelect: (value: WorkloadProfile) => void
  title: string
  description: string
}) {
  return (
    <label className={`profile-option ${selected ? 'selected' : ''}`}>
      <input type="radio" name="profile" value={value} checked={selected} onChange={() => onSelect(value)} />
      <span className="profile-radio" aria-hidden="true" />
      <span><strong>{title}</strong><small>{description}</small></span>
    </label>
  )
}

function SubmissionError({ error }: { error: Error }) {
  const apiError = error instanceof ApiError ? error : null
  return (
    <div className="form-error" role="alert">
      <strong>Environment could not be created</strong>
      <p>{error.message}</p>
      {apiError?.requestId ? <small>Request ID: {apiError.requestId}</small> : null}
    </div>
  )
}
