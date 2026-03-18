import type {
  FullConfig,
  FullResult,
  Reporter,
  Suite,
  TestCase,
  TestResult,
} from '@playwright/test/reporter';

type NdjsonEvent =
  | { type: 'begin'; suites: string[] }
  | { type: 'testBegin'; suite: string; test: string }
  | {
    type: 'testEnd';
    suite: string;
    test: string;
    passed: boolean;
    stdout: string;
    stderr: string;
    errors: string[];
    durationMs: number;
  }
  | { type: 'end' };

function emit(event: NdjsonEvent): void {
  process.stdout.write(JSON.stringify(event) + '\n');
}

const flattenSuites = (suite: Suite): string[] => {
  return [suite.title, ...suite.suites.flatMap(s => flattenSuites(s))]
}

class NdjsonReporter implements Reporter {
  onBegin(_config: FullConfig, suite: Suite): void {
    emit({ type: 'begin', suites: flattenSuites(suite) });
  }

  onTestBegin(test: TestCase): void {
    emit({
      type: 'testBegin',
      suite: test.parent.title,
      test: test.title,
    });
  }

  onTestEnd(test: TestCase, result: TestResult): void {
    const errors = result.errors.map((e) => e.message ?? String(e));
    const stdout = result.stdout
      .map((s) => (typeof s === 'string' ? s : s.toString()))
      .join('');
    const stderr = result.stderr
      .map((s) => (typeof s === 'string' ? s : s.toString()))
      .join('');

    emit({
      type: 'testEnd',
      suite: test.parent.title,
      test: test.title,
      passed: result.status === 'passed',
      stdout,
      stderr,
      errors,
      durationMs: result.duration,
    });
  }

  onEnd(_result: FullResult): void {
    emit({ type: 'end' });
  }
}

export default NdjsonReporter;
