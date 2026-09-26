#!/usr/bin/env python3
import os
import json
import argparse
from bs4 import BeautifulSoup

def parse_enrollment(dir_path, subject_code, subject_name, exam_round, exams_config, default_room='แจ้งก่อนวันสอบ', default_seat='แจ้งก่อนวันสอบ'):
    """
    Parses HTML-formatted .xls enrollment files from dir_path
    and generates a list of seating records.
    """
    seats = []
    files = sorted([f for f in os.listdir(dir_path) if f.endswith('.xls')])

    for fname in files:
        path = os.path.join(dir_path, fname)
        with open(path, 'r', encoding='windows-874', errors='ignore') as f:
            soup = BeautifulSoup(f.read(), 'html.parser')
        table = soup.find('table')
        if not table:
            continue
        
        sec_num = ''
        for tr in table.find_all('tr'):
            text = tr.get_text()
            if 'กลุ่มที่' in text:
                sec_num = text.split('กลุ่มที่')[1].strip().split()[0]
                sec_num = ''.join(c for c in sec_num if c.isdigit())
                break
                
        for tr in table.find_all('tr'):
            cols = [td.get_text(strip=True) for td in tr.find_all(['td', 'th'])]
            if len(cols) >= 5:
                std_id = cols[1]
                branch = cols[4]
                clean_id = std_id.replace('-', '').strip()
                if len(clean_id) == 10 and clean_id.isdigit():
                    for cfg in exams_config:
                        if 'section' in cfg and str(cfg['section']) != str(sec_num):
                            continue
                        room_val = cfg.get('room') if cfg.get('room') else default_room
                        seat_val = cfg.get('seat') if cfg.get('seat') else default_seat
                        seats.append({
                            'Sheet': cfg.get('sheet', ''),
                            'Date': cfg.get('date', ''),
                            'Time': cfg.get('time', ''),
                            'Room': room_val,
                            'Subject': subject_code,
                            'SubjectName': subject_name,
                            'Section': sec_num,
                            'StudentID': clean_id,
                            'Seat': seat_val,
                            'Note': cfg.get('note', ''),
                            'ExamRound': exam_round,
                            'Branch': branch,
                            'Labels': cfg.get('labels', []),
                            'CustomID': cfg.get('custom_id', '')
                        })
    return seats

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Parse enrollment XLS files into JSON seating data.')
    parser.add_argument('--dir', required=True, help='Directory containing .xls files')
    parser.add_argument('--out', required=True, help='Output JSON file path')
    parser.add_argument('--subject', required=True, help='Subject code e.g. CP422011')
    parser.add_argument('--subject-name', required=True, help='Subject name e.g. Introduction to Computer Networking')
    parser.add_argument('--round', default='mid_1_2569', help='Exam round ID e.g. mid_1_2569')
    parser.add_argument('--mapping', default='data/enrollment_mapping.json', help='Path to mapping JSON file')
    parser.add_argument('--default-room', default='แจ้งก่อนวันสอบ', help='Default room text when unspecified (e.g. ไม่ได้ระบุ)')
    parser.add_argument('--default-seat', default='แจ้งก่อนวันสอบ', help='Default seat text when unspecified (e.g. ไม่ได้ระบุ)')
    args = parser.parse_args()

    configs = []
    if os.path.exists(args.mapping):
        with open(args.mapping, 'r', encoding='utf-8') as f:
            mappings = json.load(f)
        configs = mappings.get(args.subject, [])
    else:
        print(f'Warning: Mapping file {args.mapping} not found.')

    res = parse_enrollment(args.dir, args.subject, args.subject_name, args.round, configs, args.default_room, args.default_seat)
    print(f'Parsed {len(res)} seating entries for {args.subject}.')
    os.makedirs(os.path.dirname(args.out), exist_ok=True)
    with open(args.out, 'w', encoding='utf-8') as f:
        json.dump(res, f, ensure_ascii=False, indent=2)
