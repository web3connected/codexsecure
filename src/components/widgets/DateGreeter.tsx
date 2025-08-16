'use client'

import React from 'react';
import { format, getHours } from 'date-fns';

interface Holiday {
    [key: string]: string;
}

const getHolidaysForYear = (year: number): Holiday => {
    const holidays: Holiday = {
        [`${year}-01-01`]: '🎉 Happy New Year\'s Day!',
        [`${year}-02-02`]: '🦔 Happy Groundhog Day!',
        [`${year}-02-12`]: '🎩 Lincoln\'s Birthday',
        [`${year}-02-14`]: '💘 Happy Valentine\'s Day!',
        [`${year}-03-17`]: '🍀 Happy St. Patrick\'s Day!',
        [`${year}-04-01`]: '🤪 Happy April Fool\'s Day!',
        [`${year}-04-22`]: '🌍 Happy Earth Day!',
        [`${year}-06-14`]: '🇺🇸 Happy Flag Day!',
        [`${year}-07-04`]: '🎆 Happy Independence Day!',
        [`${year}-09-11`]: '🇺🇸 Patriot Day - Remember 9/11',
        [`${year}-10-16`]: '👔 Happy Boss\'s Day!',
        [`${year}-10-31`]: '🎃 Happy Halloween!',
        [`${year}-11-01`]: '😇 Happy All Saints\' Day',
        [`${year}-11-11`]: '🎖️ Happy Veterans Day!',
        [`${year}-12-24`]: '🎄 Merry Christmas Eve!',
        [`${year}-12-25`]: '🎅 Merry Christmas!',
        [`${year}-12-26`]: '🕯️ Happy Kwanzaa!',
        [`${year}-12-31`]: '🥂 Happy New Year\'s Eve!',
    };

    return holidays;
};

const DateGreeter: React.FC = () => {
    const now = new Date();
    const currentDate = format(now, 'yyyy-MM-dd');
    const year = now.getFullYear();
    const hour = getHours(now);
    const dayName = format(now, 'EEE');
    const dateFormatted = format(now, 'MMM d, yyyy');

    const getTimeBasedGreeting = (hour: number): string => {
        if (hour >= 5 && hour < 12) return '🌅 Good Morning';
        if (hour >= 12 && hour < 17) return '☀️ Good Afternoon';
        if (hour >= 17 && hour < 21) return '🌇 Good Evening';
        return '🌙 Good Night';
    };

    const holidays = getHolidaysForYear(year);
    const holidayGreeting = holidays[currentDate];

    return (
        <div>
            {holidayGreeting || `${getTimeBasedGreeting(hour)}: ${dayName} - ${dateFormatted}`}
        </div>
    );
};

export default DateGreeter;