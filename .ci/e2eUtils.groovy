import groovy.transform.Field

@Field private workersStatus = [:]

def init(workersStatus) {
    this.workersStatus = workersStatus
}

def runE2ETests(Map args = [:]) {
    def parallelTasks = [:]
    if (!args.selectedSuites?.trim()) {
        log(level: 'DEBUG', text: "Iterate through existing test suites")
        args.testMatrix['SUITES'].each { item ->
            parallelTasks += convertSuiteToTasks(
                item: item,
                githubCheckSha1: args.githubCheckSha1 ?: '',
                githubCheckRepo: args.githubCheckRepo ?: '',
                runAsMainBranch: args.runAsMainBranch,
                amiSuffix: args.amiSuffix ?: 'main',
                destroyTestRunner: args.destroyTestRunner ?: false
            )
        }
    } else {
        log(level: 'DEBUG', text: "Iterate through the comma-separated test suites (${args.selectedSuites}), comparing with the existing test suites")
        args.selectedSuites?.split(',')?.each { selectedSuite ->
            args.testMatrix['SUITES'].findAll { selectedSuite.trim() == it.suite }.each { item ->
                parallelTasks += convertSuiteToTasks(
                    item: item,
                    githubCheckSha1: args.githubCheckSha1 ?: '',
                    githubCheckRepo: args.githubCheckRepo ?: '',
                    runAsMainBranch: args.runAsMainBranch,
                    amiSuffix: args.amiSuffix ?: 'main',
                    destroyTestRunner: args.destroyTestRunner ?: false
                )
            }
        }
    }
    parallel(parallelTasks)
}

def convertSuiteToTasks(Map args = [:]) {
    def parallelTasks = [:]
    def suite = args.item.suite
    def platforms = args.item.platforms

    // Predefine the remote provider to use the already provisioned stack VM.
    // Each suite or scenario in the CI test suite would be able to define its own provider
    // (i.e. docker). If empty, remote will be used as fallback
    def suiteProvider = args.item.provider
    if (!suiteProvider || suiteProvider?.trim() == '') {
        suiteProvider = 'remote'
    }

    args.item.scenarios.each { scenario ->
        def name = scenario.name
        def platformsValue = platforms

        def scenarioProvider = scenario.provider
        // if the scenario does not set its own provider, use suite's provider
        if (scenarioProvider?.trim() == '') {
            scenarioProvider = suiteProvider
        }
        
        if (scenario.platforms?.size() > 0) {
            // scenario platforms take precedence over suite platforms, overriding them
            platformsValue = scenario.platforms
        }
        def pullRequestFilter = scenario.containsKey('pullRequestFilter') ? scenario.pullRequestFilter : ''
        def tags = scenario.tags
        platformsValue.each { rawPlatform ->
            // platform is not space based, so let's ensure no extra spaces can cause misbehaviours.
            def platform = rawPlatform.trim()
            log(level: 'INFO', text: "Adding ${suite}:${platform}:${tags} test suite to the build execution")
            def machineInfo = getMachineInfo(platform)
            def stageName = "${suite}_${platform}_${tags}"
            parallelTasks["${stageName}"] = generateFunctionalTestStep(
                name: "${name}",
                platform: platform,
                provider: scenarioProvider,
                suite: "${suite}",
                tags: "${tags}",
                pullRequestFilter: "${pullRequestFilter}",
                machine: machineInfo,
                stageName: stageName,
                runAsMainBranch: args.runAsMainBranch,
                amiSuffix: args.amiSuffix,
                githubCheckSha1: args.githubCheckSha1,
                githubCheckRepo: args.githubCheckRepo,
                destroyTestRunner: args.destroyTestRunner                    
            )
        }
    }
    return parallelTasks
}

def checkRebuildAmis() {
    dir("${BASE_DIR}") {
        def ami_regexps = [ "^.ci/ansible/.*", "^.ci/packer/.*"]
        setEnvVar("REBUILD_AMIS", isGitRegionMatch(patterns: tests_regexps, shouldMatchAll: false))
    }
}

// this function evaluates whether the test and AMIs stages must be executed
def checkSkipTests() {
    dir("${BASE_DIR}") {

        // if only docs changed means no tests are run
        if (isGitRegionMatch(patterns: [ '.*\\.md' ], shouldMatchAll: true)) {
            setEnvVar("SKIP_TESTS", true)            
            return
        }

        // patterns for all places that should trigger a full build
        def tests_regexps = [ 
            "^e2e/_suites/fleet/.*", 
            "^e2e/_suites/helm/.*", 
            "^e2e/_suites/kubernetes-autodiscover/.*", 
            "^.ci/.*", 
            "^cli/.*", 
            "^e2e/.*\\.go", 
            "^internal/.*\\.go" ]
        // def ami_regexps = [ "^.ci/ansible/.*", "^.ci/packer/.*"]
        setEnvVar("SKIP_TESTS", !isGitRegionMatch(patterns: tests_regexps, shouldMatchAll: false))        
    }
}

/*
 * Runs the Make build at the CI, executing the closure in the context of Ansible + AWS
 */
def ciBuild(Closure body){
    withEnv([
            "SSH_KEY=${E2E_SSH_KEY}"
    ]) {
        def awsProps = getVaultSecret(secret: "${AWS_PROVISIONER_SECRET}")
        def awsAuthObj = awsProps?.data
        withEnv([
                "ANSIBLE_CONFIG=${env.REAL_BASE_DIR}/.ci/ansible/ansible.cfg",
                "ANSIBLE_HOST_KEY_CHECKING=False",
        ]){
            withVaultToken(){
                withEnvMask(vars: [
                        [var: "AWS_ACCESS_KEY_ID", password: awsAuthObj.access_key],
                        [var: "AWS_SECRET_ACCESS_KEY", password: awsAuthObj.secret_key]
                ]) {
                    withOtelEnv() {
                        retryWithSleep(retries: 3, seconds: 5, backoff: true){
                            sh("make -C .ci setup-env") // make sure the environment is created
                        }
                        body()
                    }
                }
            }
        }
    }
}

def getNodeIp(nodeType){
    return sh(label: "Get IP address of the ${nodeType}", script: "cat ${REAL_BASE_DIR}/.ci/.${nodeType}-host-ip", returnStdout: true)
}

def getMachineInfo(platform){
    def machineYaml = readYaml(file: "${env.REAL_BASE_DIR}/.ci/.e2e-platforms.yaml")
    def machines = machineYaml['PLATFORMS']
    log(level: 'INFO', text: "getMachineInfo: machines.get(platform)=${machines.get(platform)}")
    return machines.get(platform)
}

/*
 * Sends out notification of the build result to Slack
 */
def doNotifyBuildResult(Map args = [:]) {
    def doSlackNotify = true // always try to notify on failures
    def githubCheckStatus = 'FAILURE'
    if (currentBuild.currentResult == 'SUCCESS') {
        githubCheckStatus = 'SUCCESS'
        doSlackNotify = args.slackNotify // if the build status is success, read the parameter
    }

    this.githubCheckNotify(
        status: githubCheckStatus,
        githubCheckRepo: args.githubCheckRepo,
        githubCheckSha1: args.githubCheckSha1,
        githubCheckName: args.githubCheckName
    )

    def testsSuites = args.runTestsSuites?.trim() ?: "All suites"    
    def channels = args.slackChannel?.trim() ?: "observablt-bots"    

    def header = "*Test Suite*: ${testsSuites}"
    notifyBuildResult(analyzeFlakey: true,
            jobName: getFlakyJobName(withBranch: "${env.JOB_BASE_NAME}"),
            prComment: true,
            slackHeader: header,
            slackChannel: "${channels}",
            slackComment: true,
            slackNotify: doSlackNotify)
}

/**
 Notify the GitHub check of the parent stream
 **/
def githubCheckNotify(Map args = [:]) {
    if (args.githubCheckName?.trim() && args.githubCheckRepo?.trim() && args.githubCheckSha1?.trim()) {
        githubNotify context: "${args.githubCheckName}",
                description: "${args.githubCheckName} ${args.status?.toLowerCase()}",
                status: "${args.status}",
                targetUrl: "${env.RUN_DISPLAY_URL}",
                sha: "${args.githubCheckSha1}", 
                account: 'elastic', 
                repo: args.githubCheckRepo, 
                credentialsId: env.JOB_GIT_CREDENTIALS
    }
}
/**
* 
*
*/
def generateFunctionalTestStep(Map args = [:]) {
    def name = args.get('name')
    def name_normalize = name.replace(' ', '_')
    def platform = args.get('platform')
    def provider = args.get('provider')
    def suite = args.get('suite')
    def tags = args.get('tags')
    def pullRequestFilter = args.get('pullRequestFilter')?.trim() ?: ''
    def machine = args.get('machine')
    def stageName = args.get('stageName')
    //TODO
    def ami_suffix = args.amiSuffix.trim() ?: 'main'
    def runAsMainBranch = args.runAsMainBranch ?: false
    def destroyTestRunner = args.destroyTestRunner ?: false
    

    if (isPR() || isUpstreamTrigger(filter: 'PR-')) {
        // when the "Run_As_Main_Branch" param is disabled, we will honour the PR filters, which
        // basically exclude some less frequent platforms or operative systems. If the user enabled
        // this param, the pipeline will remove the filters from the test runner.
        if (!runAsMainBranch) {
            tags += pullRequestFilter
        }        
    }

    def goArch = platform.contains("arm64") ? "arm64" : "amd64"

    // sanitize tags to create the file
    def sanitisedTags = tags.replaceAll("\\s","_")
    sanitisedTags = sanitisedTags.replaceAll("~","")
    sanitisedTags = sanitisedTags.replaceAll("@","")

    def githubCheckSha1 = args.githubCheckSha1?.trim() ?: ''
    def githubCheckRepo = args.githubCheckRepo?.trim() ?: ''

    // Setup environment for platform
    def envContext = []
    envContext.add("PROVIDER=${provider}")
    envContext.add("GITHUB_CHECK_SHA1=${githubCheckSha1}")
    envContext.add("GITHUB_CHECK_REPO=${githubCheckRepo}")
    envContext.add("SUITE=${suite}")
    envContext.add("TAGS=${tags}")
    envContext.add("REPORT_PREFIX=${suite}_${platform}_${sanitisedTags}")
    envContext.add("ELASTIC_APM_GLOBAL_LABELS=branch_name=${BRANCH_NAME},build_pr=${isPR()},build_id=${env.BUILD_ID},go_arch=${goArch},beat_version=${env.BEAT_VERSION},elastic_agent_version=${env.ELASTIC_AGENT_VERSION},stack_version=${env.STACK_VERSION}")
    // VM characteristics
    envContext.add("NODE_LABEL=${platform}")
    envContext.add("NODE_IMAGE=${machine.image}-${ami_suffix}")
    envContext.add("NODE_INSTANCE_ID=${env.BUILD_URL}_${platform}_${suite}_${tags}")
    envContext.add("NODE_INSTANCE_TYPE=${machine.instance_type}")
    envContext.add("NODE_SHELL_TYPE=${machine.shell_type}")
    envContext.add("NODE_USER=${machine.username}")

    return {
        // Set the worker as flaky for the time being, this will be changed in the finally closure.
        setFlakyWorker(stageName)
        retryWithNode(labels: 'ubuntu-20.04 && gobld/machineType:e2-small', forceWorkspace: true, forceWorker: true, stageName: stageName){
            try {
                deleteDir()
                dir("${env.REAL_BASE_DIR}") {
                    unstash 'sourceEnvModified'
                    withEnv(envContext) {
                        // This step will help to send the APM traces to the
                        // withOtelEnv is the one that uses the APM service defined by the Otel Jenkins plugin.
                        // withAPMEnv uses Vault to prepare the context.
                        // IMPORTANT: withAPMEnv is now the one in used since withOtelEnv uses a specific Opentelemetry Collector at the moment.
                        // TODO: This will need to be integrated into the provisioned VMs
                        withAPMEnv() {
                            // we are separating the different test phases to avoid recreating
                            ciBuild() {
                                sh(label: 'Start node', script: "make -C .ci provision-node")
                            }

                            // make goal to run the tests, which is platform-dependant
                            def runCommand = "run-tests"

                            if (platform.contains("windows")) {
                                runCommand = "run-tests-win"
                                // Ansible wait_for module is not enough to mitigate the timeout
                                log(level: 'DEBUG', text: "Sleeping 300 seconds on Windows so that SSH is accessible in the remote instance.")
                                sleep(300)
                            }
                            if (!machine.dependencies_installed) {
                                ciBuild() {
                                    retryWithSleep(retries: 3, seconds: 5, backoff: true){
                                        sh(label: 'Configure node for testing', script: "make -C .ci setup-node")
                                    }
                                }
                            }
                            ciBuild() {
                                sh(label: 'Run tests in the node', script: "make -C .ci ${runCommand}")
                            }
                        }
                    }
                }
            } finally {
                withEnv(envContext) {
                    dir("${env.REAL_BASE_DIR}") {
                        // If it reaches this point then the CI worker is most likely behaving correctly
                        // there is still a chance things might fail afterwards, but this is just the finally
                        // section so we could say we are good to go.
                        // It runs after dir so if the worker is gone the an error will be thrown regarding
                        // the dir cannot be accessed in the existing none worker.
                        unsetFlakyWorker(stageName)
                        def testRunnerIP = getNodeIp("node")
                        sh "mkdir -p outputs/${testRunnerIP} || true"
                        ciBuild() {
                            sh(label: 'Fetch tests reports from node', script: "make -C .ci fetch-test-reports")
                        }
                        sh "ls -l outputs/${testRunnerIP}"
                        if (!destroyTestRunner) {
                            log(level: 'INFO', text: "Cloud instance won't be destroyed after the build. Please SSH into the test runner machine on ${testRunnerIP}.")
                        } else {
                            log(level: 'INFO', text: "Destroying Cloud instance")
                            ciBuild() {
                                retryWithSleep(retries: 3, seconds: 5, backoff: true){
                                    sh(label: 'Destroy node', script: "make -C .ci destroy-node")
                                }
                            }
                        }
                        withEnv([
                                "ARCHITECTURE=${goArch}",
                                "CUCUMBER_REPORTS_PATH=${env.REAL_BASE_DIR}/outputs/${testRunnerIP}",
                                "PLATFORM=${platform}",
                                "SUITE=${suite}",
                                "TAGS=${tags}",
                        ]){
                            retryWithSleep(retries: 3, seconds: 5, backoff: true){
                                dockerLogin(secret: "${DOCKER_ELASTIC_SECRET}", registry: "${DOCKER_REGISTRY}")
                                sh(script: ".ci/scripts/generate-cucumber-reports.sh", label: "generate-cucumber-reports.sh")
                            }
                        }
                        junit2otel(traceName: 'junit-e2e-tests', allowEmptyResults: true, keepLongStdio: true, testResults: "outputs/${testRunnerIP}/TEST-*.xml")
                        archiveArtifacts allowEmptyArchive: true,
                                artifacts: "outputs/${testRunnerIP}/TEST-*.xml, outputs/${testRunnerIP}/TEST-*.json, outputs/${testRunnerIP}/TEST-*.json.html"
                    }
                }
            }
        }
    }
}

def retryWithNode(Map args = [:], Closure body) {
    try {
        incrementRetries(args.stageName)
        withNode(args){
            body()
        }
    } catch (err) {
        log(level: 'WARN', text: "Stage '${args.stageName}' failed, let's analyse if it's a flaky CI worker.")
        if (isFlakyWorker(args.stageName) && isRetryAvailable(args.stageName)) {
            log(level: 'INFO', text: "Rerun '${args.stageName}' in a new worker.")
            retryWithNode(args) {
                body()
            }
        } else {
            error("Error '${err.toString()}'")
        }
    }
}

def isFlakyWorker(stageName) {
    if (workersStatus.containsKey(stageName)) {
        return !workersStatus.get(stageName).get('status', true)
    }
    return false
}

def isRetryAvailable(stageName) {
    return workersStatus.get(stageName).get('retries', 2) < 2
}

def incrementRetries(stageName) {
    if (workersStatus.containsKey(stageName)) {
        def current = workersStatus[stageName].get('retries', 0)
        workersStatus[stageName].retries = current + 1
    } else {
        setFlakyWorker(stageName)
        workersStatus[stageName].retries = 1
    }
}

def setFlakyWorker(stageName) {
    if (workersStatus.containsKey(stageName)) {
        workersStatus[stageName].status = false
    } else {
        workersStatus[stageName] = [ status: false ]
    }
}

def unsetFlakyWorker(stageName) {
    workersStatus[stageName].status = true
}

def destroyStack(Map args = [:]) {
    def stackMachine = getMachineInfo('stack')
    if (!args.destroyCloudResources) {
        def stackRunnerIP = getNodeIp('stack')
        log(level: 'DEBUG', text: "Stack instance won't be destroyed after the build. Please SSH into the stack machine on ${stackRunnerIP}")
    } else {
        dir("${env.REAL_BASE_DIR}") {
            ciBuild() {
                retryWithSleep(retries: 3, seconds: 5, backoff: true){
                    sh(label: 'Destroy stack node', script: "make -C .ci destroy-stack")
                }
            }
        }
    }
}

return this