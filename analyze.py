import sys
import argparse
import json


def cnt_on_rule_id(issues, rule_id):
    return len([issue for issue in issues if issue['rule_id'] == rule_id])


def write_issue(f, issue, idx):
    f.write('Issue %d\\n' % idx)
    for k, v in issue.items():
        f.write('|%s|%s|\\n' % (k, v))


def analyze(js, formatted_issues_f):
    issues = js['Issues']
    if not issues:
        print("Security check: no security issue detected")
        return 0

    for issue in issues:
        f = issue['file']
        f = '/'.join(f.split('/')[2:])
        issue['file'] = f

    must_fix = []
    better_fix = []
    for issue in issues:
        if issue['severity'] == 'HIGH':
            must_fix.append(issue)
        else:
            better_fix.append(issue)


    with open(formatted_issues_f, 'w') as f:
        idx = 1
        f.write('\\n*Must fix issues*\\n')
        print('======== Must fix the potential security issues ========')
        for issue in must_fix:
            print(json.dumps(issue, indent=4))
            write_issue(f, issue, idx)
            idx += 1

        f.write('\\n----\\n*Optinal fix issues*\\n')
        print('======== Optional to fix the potential security issues ========')
        for issue in better_fix:
            print(json.dumps(issue, indent=4))
            write_issue(f, issue, idx)
            idx += 1

    if must_fix:
        return 1
    else:
        return 0


def parse_args_or_exit(argv=None):
    """
    Parse command line options
    """
    parser = argparse.ArgumentParser(description="Analyze security check result")
    parser.add_argument("-i", metavar="check_result",
            dest="check_result", help="json file of check result")
    parser.add_argument("issues", metavar="issues",
            help="formatted issues")

    args = parser.parse_args(argv)

    return args

def main(argv):
    args = parse_args_or_exit(argv)
    check_result = args.check_result
    with open(args.check_result) as f:
        js = json.load(f)
        sys.exit(analyze(js, args.issues))

if __name__ == '__main__':
    main(sys.argv[1:])
