// Check out https://github.com/showheroes/jenkins-shared-libs
@Library('shRelease') _

def dcr = "dcr.jenkins:5000"
def dockerImageName = "sh-oec"
def gar_locations = ["us"]
def gar_repo = "viralize-143916/infra"

def releaseConfig = [
    "default_release": "minor"
]

def buildImage(name, dcr) {
    sh """
        docker build --network host \
        --cache-to type=inline \
        --build-arg ICINGA2_VERSION=${env.ICINGA2_VERSION} \
        --build-arg version=${env.RELEASE_TAG} \
        -t ${dcr}/${name}:${GIT_COMMIT} \
        -t ${dcr}/${name}:${CURRENT_BRANCH} \
        --cache-from ${dcr}/${name}:latest \
        --cache-from ${dcr}/${name}:${CURRENT_BRANCH} \
        --cache-from ${dcr}/${name}:${SRC_BRANCH} \
        --push .
    """
    if (env.CURRENT_BRANCH == 'master') {
        sh "crane tag ${dcr}/${name}:${GIT_COMMIT} latest --insecure"
    }
}

pipeline {
    // Check out https://github.com/showheroes/jenkins-agents
    agent { label 'jnlp_dind_v2' }
    environment {
        CURRENT_BRANCH = "${env.CHANGE_BRANCH ?: env.BRANCH_NAME}"
        SRC_BRANCH = shRelease.getPullRequestInfo().head.ref.trim()
        GITHUB_TOKEN = credentials('SHBOT_GITHUB_RELEASE_TOKEN')
        RELEASE_TAG = shRelease.getReleaseTag(releaseConfig)
    }

    stages {
       stage('Docker Build') {
           steps {
               // Build the Docker image
               script {
                    def lastRelease = shRelease.getLastRelease()
                    echo "Last stable release is: $lastRelease"
                    def prID = shRelease.getPullRequestID()
                    echo "PR ID is: $prID"
                    buildImage(dockerImageName, dcr)
               }
           }
       }
        stage('Release Candidate') {
            when {
                allOf {
                    changeRequest();
                }
            }
            steps {
                script {
                    shRelease.releaseCandidate(env.RELEASE_TAG)
                    shRelease.copyDockerImage(
                        source_image: "${dcr}/${dockerImageName}:${GIT_COMMIT}",
                        target_image: dockerImageName,
                        gar_locations: gar_locations,
                        gar_repo: gar_repo,
                        target_tags: [env.RELEASE_TAG]
                    )
                }
            }
        }
        stage('Release Stable') {
            when {
                allOf {
                    branch 'master'
                }
            }
            steps {
                script {
                    shRelease.releaseStable(env.RELEASE_TAG)
                    shRelease.copyDockerImage(
                        source_image: "${dcr}/${dockerImageName}:${GIT_COMMIT}",
                        target_image: dockerImageName,
                        gar_locations: gar_locations,
                        gar_repo: gar_repo,
                        target_tags: [env.RELEASE_TAG, 'latest']
                    )
                }
            }
        }
    }
}
